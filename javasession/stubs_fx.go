package javasession

// Not implemented yet (effects area): these do nothing. Generated with tools/stubgen from
// player.Session; delete a method here when it gets a real implementation.

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
)

// Methods the Java session does not implement yet: they do nothing. Generated with
// tools/stubgen from player.Session; delete a line when the method gets a real implementation.

func (*Session) PlaySound(_ world.Sound, _ mgl64.Vec3)                       {}
func (*Session) SendEffect(_ effect.Effect)                                  {}
func (*Session) SendEffectRemoval(_ effect.Type)                             {}
func (*Session) SendSpeed(_ float64)                                         {}
func (*Session) ViewBlockAction(_ cube.Pos, _ world.BlockAction)             {}
func (*Session) ViewEmote(_ world.Entity, _ uuid.UUID)                       {}
func (*Session) ViewEntityAnimation(_ world.Entity, _ world.EntityAnimation) {}
func (*Session) ViewEntityWake(_ world.Entity)                               {}
func (*Session) ViewParticle(_ mgl64.Vec3, _ world.Particle)                 {}
func (*Session) ViewSleepingPlayers(_ int, _ int)                            {}
func (*Session) ViewSound(_ mgl64.Vec3, _ world.Sound)                       {}
