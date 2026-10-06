package javasession

import (
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	v777 "github.com/ezchr/go-mc/java/v777"
	"github.com/go-gl/mathgl/mgl64"
)

// Java entity type ids (minecraft:entity_type protocol ids in Mojang's registries.json).
const entityTypePlayer = 159

// entityID returns the Java entity id this client knows e by.
func (s *Session) entityID(e world.Entity) (int32, bool) {
	if e.H() == s.ent {
		return selfEntityID, true
	}
	s.entMu.Lock()
	defer s.entMu.Unlock()
	id, ok := s.entityIDs[e.H()]
	return id, ok
}

func (s *Session) addEntityID(e world.Entity) int32 {
	s.entMu.Lock()
	defer s.entMu.Unlock()
	if id, ok := s.entityIDs[e.H()]; ok {
		return id
	}
	s.nextEntityID++
	if s.nextEntityID == selfEntityID {
		s.nextEntityID++
	}
	s.entityIDs[e.H()] = s.nextEntityID
	return s.nextEntityID
}

func (s *Session) removeEntityID(e world.Entity) (int32, bool) {
	s.entMu.Lock()
	defer s.entMu.Unlock()
	id, ok := s.entityIDs[e.H()]
	delete(s.entityIDs, e.H())
	return id, ok
}

// tabName is a name the Java client accepts in the tab list (at most 16 characters).
func tabName(n string) string {
	r := []rune(n)
	if len(r) > 16 {
		r = r[:16]
	}
	return string(r)
}

// ViewEntity shows an entity that came into view.
func (s *Session) ViewEntity(e world.Entity) {
	if e.H() == s.ent {
		return
	}
	p, ok := e.(*player.Player)
	if !ok {
		s.viewOtherEntity(e)
		return
	}
	id := s.addEntityID(e)
	u := p.UUID()

	// Tab list entry first: the client needs the profile to spawn a player entity.
	w := s.packet()
	w.Byte(1<<0 | 1<<2 | 1<<3 | 1<<4) // ADD_PLAYER, UPDATE_GAME_MODE, UPDATE_LISTED, UPDATE_LATENCY
	w.VarInt(1)
	w.UUID(u)
	w.String(tabName(p.Name()))
	// Java players' signed textures show their skin; without properties the client picks a
	// default skin from the UUID.
	props := profileProperties(u)
	w.VarInt(int32(len(props)))
	for _, pr := range props {
		w.String(pr.Name)
		w.String(pr.Value)
		w.Bool(pr.Signature != "")
		if pr.Signature != "" {
			w.String(pr.Signature)
		}
	}
	w.VarInt(gameModeID(p.GameMode()))
	w.Bool(true)
	w.VarInt(int32(p.Latency() / time.Millisecond))
	s.queue(v777.ClientboundPlayPlayerInfoUpdate, w)

	pos, rot := p.Position(), p.Rotation()
	w = s.packet()
	w.VarInt(id)
	w.UUID(u)
	w.VarInt(entityTypePlayer)
	w.Float64(pos[0])
	w.Float64(pos[1])
	w.Float64(pos[2])
	w.LpVec3(0, 0, 0)
	w.Angle(float32(rot.Pitch()))
	w.Angle(float32(rot.Yaw()))
	w.Angle(float32(rot.Yaw())) // head
	w.VarInt(0)
	s.queue(v777.ClientboundPlayAddEntity, w)
}

// HideEntity removes an entity that left view.
func (s *Session) HideEntity(e world.Entity) {
	if e.H() == s.ent {
		return
	}
	id, ok := s.removeEntityID(e)
	if !ok {
		return
	}
	w := s.packet()
	w.VarInt(1)
	w.VarInt(id)
	s.queue(v777.ClientboundPlayRemoveEntities, w)
	if p, ok := e.(*player.Player); ok {
		w = s.packet()
		w.VarInt(1)
		w.UUID(p.UUID())
		s.queue(v777.ClientboundPlayPlayerInfoRemove, w)
	}
}

// ViewEntityMovement moves another entity to an absolute position.
func (s *Session) ViewEntityMovement(e world.Entity, pos mgl64.Vec3, rot cube.Rotation, onGround bool) {
	if e.H() == s.ent {
		return // the client moves itself
	}
	s.positionSync(e, pos, rot, onGround)
}

// ViewEntityDisplacement is a server-made move (knockback correction, pushing).
func (s *Session) ViewEntityDisplacement(e world.Entity, pos mgl64.Vec3, rot cube.Rotation, onGround bool) {
	if e.H() == s.ent {
		s.teleport(pos[0], pos[1], pos[2], float32(rot.Yaw()), float32(rot.Pitch()))
		return
	}
	s.positionSync(e, pos, rot, onGround)
}

// ViewEntityTeleport moves an entity instantly.
func (s *Session) ViewEntityTeleport(e world.Entity, pos mgl64.Vec3) {
	rot := e.Rotation()
	if e.H() == s.ent {
		s.teleport(pos[0], pos[1], pos[2], float32(rot.Yaw()), float32(rot.Pitch()))
		return
	}
	s.positionSync(e, pos, rot, false)
}

func (s *Session) positionSync(e world.Entity, pos mgl64.Vec3, rot cube.Rotation, onGround bool) {
	id, ok := s.entityID(e)
	if !ok {
		return
	}
	w := s.packet()
	w.VarInt(id)
	w.VarInt(0) // PositionPath.LINEAR
	w.Float64(pos[0])
	w.Float64(pos[1])
	w.Float64(pos[2])
	w.Float32(float32(rot.Yaw()))
	w.Float32(float32(rot.Pitch()))
	w.Bool(onGround)
	s.queue(v777.ClientboundPlayEntityPositionSync, w)
	w = s.packet()
	w.VarInt(id)
	w.Angle(float32(rot.Yaw()))
	s.queue(v777.ClientboundPlayRotateHead, w)
}

// ViewEntityVelocity sets an entity's motion; for the player itself this is knockback.
func (s *Session) ViewEntityVelocity(e world.Entity, vel mgl64.Vec3) {
	id, ok := s.entityID(e)
	if !ok {
		return
	}
	w := s.packet()
	w.VarInt(id)
	w.LpVec3(vel[0], vel[1], vel[2])
	s.queue(v777.ClientboundPlaySetEntityMotion, w)
}

// ViewEntityAction plays an entity animation.
func (s *Session) ViewEntityAction(e world.Entity, a world.EntityAction) {
	id, ok := s.entityID(e)
	if !ok {
		return
	}
	switch a.(type) {
	case entity.SwingArmAction:
		if id == selfEntityID {
			return // the client already swung
		}
		// 26.3 swings with swing_animation (animate no longer has a swing action).
		w := s.packet()
		w.VarInt(id)
		w.VarInt(0) // main hand
		w.VarInt(1) // SwingAnimationType.WHACK
		w.VarInt(6) // duration in ticks (SwingAnimation.DEFAULT)
		s.queue(v777.ClientboundPlaySwingAnimation, w)
	case entity.HurtAction:
		w := s.packet()
		w.VarInt(id)
		w.Float32(float32(e.Rotation().Yaw()))
		s.queue(v777.ClientboundPlayHurtAnimation, w)
	case entity.CriticalHitAction:
		s.animate(id, 1) // CRITICAL_HIT
	case entity.EnchantedHitAction:
		s.animate(id, 2) // MAGIC_CRITICAL_HIT
	}
}

// animate sends an animate packet (26.3 actions: 0 wake up, 1 critical hit, 2 magic critical hit).
func (s *Session) animate(id int32, action byte) {
	w := s.packet()
	w.VarInt(id)
	w.Byte(action)
	s.queue(v777.ClientboundPlayAnimate, w)
}

// ViewBlockUpdate sends a changed block.
func (s *Session) ViewBlockUpdate(pos cube.Pos, b world.Block, layer int) {
	if layer != 0 {
		return // water layer changes: waterlogging fixer to come
	}
	rid := world.BlockRuntimeID(b)
	bi := blocks()
	if int(rid) >= len(bi.java) {
		return
	}
	w := s.packet()
	w.Position(pos[0], pos[1], pos[2])
	w.VarInt(int32(bi.java[rid]))
	s.queue(v777.ClientboundPlayBlockUpdate, w)
}
