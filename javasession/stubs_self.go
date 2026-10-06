package javasession

// Not implemented yet (self area): these do nothing. Generated with tools/stubgen from
// player.Session; delete a method here when it gets a real implementation.

import (
	"net"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/df-mc/dragonfly/server/player/hud"
	"github.com/df-mc/dragonfly/server/player/input"
	"github.com/df-mc/dragonfly/server/player/skin"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
)

// Methods the Java session does not implement yet: they do nothing. Generated with
// tools/stubgen from player.Session; delete a line when the method gets a real implementation.

func (*Session) AddDebugShape(_ debug.Shape)                                 {}
func (*Session) ClearInputLocks()                                            {}
func (*Session) EnableCoordinates(_ bool)                                    {}
func (*Session) EnableInstantRespawn(_ bool)                                 {}
func (*Session) HideHudElement(_ hud.Element)                                {}
func (*Session) HudElementHidden(_ hud.Element) bool                         { return false }
func (*Session) InputLocked(_ input.Lock) bool                               { return false }
func (*Session) LockInput(_ input.Lock)                                      {}
func (*Session) PlaySound(_ world.Sound, _ mgl64.Vec3)                       {}
func (*Session) RemoveAllDebugShapes()                                       {}
func (*Session) RemoveDebugShape(_ debug.Shape)                              {}
func (*Session) RemoveViewLayer(_ world.Entity)                              {}
func (*Session) SendDebugShapes(_ world.Dimension)                           {}
func (*Session) SendEffect(_ effect.Effect)                                  {}
func (*Session) SendEffectRemoval(_ effect.Type)                             {}
func (*Session) SendHudUpdates()                                             {}
func (*Session) SendInputLocks()                                             {}
func (*Session) SendSpeed(_ float64)                                         {}
func (*Session) SetHandle(_ *world.EntityHandle, _ skin.Skin)                {}
func (*Session) ShowHudElement(_ hud.Element)                                {}
func (*Session) StartShowingEntity(_ world.Entity)                           {}
func (*Session) StopShowingEntity(_ world.Entity)                            {}
func (*Session) Transfer(_ net.IP, _ int)                                    {}
func (*Session) UnlockInput(_ input.Lock)                                    {}
func (*Session) ViewBlockAction(_ cube.Pos, _ world.BlockAction)             {}
func (*Session) ViewEmote(_ world.Entity, _ uuid.UUID)                       {}
func (*Session) ViewEntityAnimation(_ world.Entity, _ world.EntityAnimation) {}
func (*Session) ViewEntityDismount(_ world.Entity, _ world.Entity)           {}
func (*Session) ViewEntityGameMode(_ world.Entity)                           {}
func (*Session) ViewEntityMount(_ world.Entity, _ world.Entity, _ bool)      {}
func (*Session) ViewEntityWake(_ world.Entity)                               {}
func (*Session) ViewLayer() *world.ViewLayer                                 { return nil }
func (*Session) ViewParticle(_ mgl64.Vec3, _ world.Particle)                 {}
func (*Session) ViewSkin(_ world.Entity)                                     {}
func (*Session) ViewSleepingPlayers(_ int, _ int)                            {}
func (*Session) ViewSound(_ mgl64.Vec3, _ world.Sound)                       {}
func (*Session) ViewTime(_ int)                                              {}
func (*Session) ViewTimeCycle(_ bool)                                        {}
func (*Session) ViewVisibility(_ world.Entity, _ world.VisibilityLevel)      {}
func (*Session) ViewWeather(_ bool, _ bool)                                  {}
func (*Session) VisibleDebugShapes() []debug.Shape                           { return nil }
