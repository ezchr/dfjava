package javasession

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	v777 "github.com/ezchr/go-mc/java/v777"
	"github.com/ezchr/go-mc/java/wire"
	"github.com/go-gl/mathgl/mgl64"
)

// inputState is the client input the session keeps between packets.
type inputState struct {
	sneaking  bool
	jumping   bool
	breaking  bool
	breakFace cube.Face
}

// Java player_action actions.
const (
	actionStartDestroy = iota
	actionChangeDestroyDirection
	actionAbortDestroy
	actionStopDestroy
	actionDropAll
	actionDropItem
	actionReleaseUseItem
	actionSwapOffhand
	actionStab
)

// Java player_input flags.
const (
	inputJump  = 16
	inputShift = 32
)

// Java player_command actions.
const (
	commandStopSleeping = iota
	commandStartSprinting
	commandStopSprinting
)

// handleInput handles the client's world interaction packets.
func (s *Session) handleInput(id int32, body []byte) (bool, error) {
	r := wire.NewReader(body)
	switch id {
	case v777.ServerboundPlayPlayerAction:
		action := r.VarInt()
		x, y, z := r.Position()
		face := cube.Face(r.Byte())
		seq := r.VarInt()
		if r.Err != nil || face > cube.FaceEast {
			return true, r.Err
		}
		pos := cube.Pos{x, y, z}
		s.do(func(tx *world.Tx, c session.Controllable) {
			// Bedrock clients drive the breaking animation with start/continue/stop and send the
			// actual break separately (Dragonfly's BreakBlock). A Java client breaks instantly in
			// creative on START, and in survival says STOP when it finished breaking.
			switch action {
			case actionStartDestroy:
				if c.GameMode().CreativeInventory() {
					c.BreakBlock(pos)
					return
				}
				c.StartBreaking(pos, face)
				s.input.breaking, s.input.breakFace = true, face
			case actionAbortDestroy:
				c.AbortBreaking()
				s.input.breaking = false
			case actionStopDestroy:
				c.FinishBreaking()
				c.BreakBlock(pos)
				s.input.breaking = false
			case actionReleaseUseItem:
				c.ReleaseItem()
			}
		})
		s.ackBlock(seq)
	case v777.ServerboundPlayUseItemOn:
		hand := r.VarInt()
		x, y, z := r.Position()
		face := cube.Face(r.VarInt())
		cx, cy, cz := r.Float32(), r.Float32(), r.Float32()
		r.Bool() // inside block
		r.Bool() // world border hit
		seq := r.VarInt()
		if r.Err != nil || face > cube.FaceEast {
			return true, r.Err
		}
		if hand == 0 {
			pos := cube.Pos{x, y, z}
			click := mgl64.Vec3{float64(cx), float64(cy), float64(cz)}
			s.do(func(tx *world.Tx, c session.Controllable) { c.UseItemOnBlock(pos, face, click) })
		}
		s.ackBlock(seq)
	case v777.ServerboundPlayUseItem:
		hand := r.VarInt()
		seq := r.VarInt()
		r.Float32()
		r.Float32()
		if r.Err != nil {
			return true, r.Err
		}
		if hand == 0 {
			s.do(func(tx *world.Tx, c session.Controllable) { c.UseItem() })
		}
		s.ackBlock(seq)
	case v777.ServerboundPlayAttack:
		target := r.VarInt()
		if r.Err != nil {
			return true, r.Err
		}
		s.do(func(tx *world.Tx, c session.Controllable) {
			e, ok := s.entityByID(tx, target)
			if !ok {
				s.log.Debug("attack: unknown target", "id", target)
				return
			}
			// AttackEntity swings the arm itself when the attack is valid.
			hit := c.AttackEntity(e)
			s.log.Debug("attack", "target", target, "hit", hit)
		})
	case v777.ServerboundPlayPunch:
		s.do(func(tx *world.Tx, c session.Controllable) { c.PunchAir() })
	case v777.ServerboundPlayInteract:
		target := r.VarInt()
		hand := r.VarInt()
		r.LpVec3()
		r.Bool() // sneaking
		if r.Err != nil {
			return true, r.Err
		}
		if hand == 0 {
			s.do(func(tx *world.Tx, c session.Controllable) {
				if e, ok := s.entityByID(tx, target); ok {
					c.UseItemOnEntity(e)
				}
			})
		}
	case v777.ServerboundPlayPlayerInput:
		flags := r.Byte()
		if r.Err != nil {
			return true, r.Err
		}
		sneak, jump := flags&inputShift != 0, flags&inputJump != 0
		s.do(func(tx *world.Tx, c session.Controllable) {
			if sneak != s.input.sneaking {
				if sneak {
					c.StartSneaking()
				} else {
					c.StopSneaking()
				}
				s.input.sneaking = sneak
			}
			if jump && !s.input.jumping {
				c.Jump()
			}
			s.input.jumping = jump
		})
	case v777.ServerboundPlayPlayerCommand:
		r.VarInt() // entity id: always the player
		action := r.VarInt()
		r.VarInt()
		if r.Err != nil {
			return true, r.Err
		}
		s.do(func(tx *world.Tx, c session.Controllable) {
			switch action {
			case commandStartSprinting:
				c.StartSprinting()
			case commandStopSprinting:
				c.StopSprinting()
			}
		})
	case v777.ServerboundPlayClientCommand:
		if action := r.VarInt(); r.Err == nil && action == 0 { // perform respawn
			s.do(func(tx *world.Tx, c session.Controllable) { c.Respawn() })
		}
	default:
		return false, nil
	}
	return true, r.Err
}

// do runs f in the player's world transaction, logging unexpected failures.
func (s *Session) do(f func(tx *world.Tx, c session.Controllable)) {
	if err := s.withPlayer(f); err != nil && !stopped(err) {
		s.log.Debug("input", "err", err)
	}
}

// ackBlock tells the client the server handled its block action up to seq, so it drops its
// prediction (and shows the server's blocks instead).
func (s *Session) ackBlock(seq int32) {
	w := s.packet()
	w.VarInt(seq)
	s.queue(v777.ClientboundPlayBlockChangedAck, w)
}

// continueBreaking is called every tick while the client holds the break key.
func (s *Session) continueBreaking(c session.Controllable) {
	if s.input.breaking {
		c.ContinueBreaking(s.input.breakFace)
	}
}

// entityByID finds an entity the client knows by its Java id.
func (s *Session) entityByID(tx *world.Tx, id int32) (world.Entity, bool) {
	s.entMu.Lock()
	var h *world.EntityHandle
	for eh, eid := range s.entityIDs {
		if eid == id {
			h = eh
			break
		}
	}
	s.entMu.Unlock()
	if h == nil {
		return nil, false
	}
	return h.Entity(tx)
}
