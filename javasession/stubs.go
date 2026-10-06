package javasession

import (
	"net"
	"time"

	"github.com/df-mc/dragonfly/server/session"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player/chat"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/df-mc/dragonfly/server/player/dialogue"
	"github.com/df-mc/dragonfly/server/player/form"
	"github.com/df-mc/dragonfly/server/player/hud"
	"github.com/df-mc/dragonfly/server/player/input"
	"github.com/df-mc/dragonfly/server/player/scoreboard"
	"github.com/df-mc/dragonfly/server/player/skin"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"golang.org/x/text/language"
)

// Methods the Java session does not implement yet: they do nothing. Generated with
// tools/stubgen from player.Session; delete a line when the method gets a real implementation.

func (*Session) AddDebugShape(_ debug.Shape) {}
func (*Session) ClearInputLocks()            {}
func (*Session) CloseDialogue()              {}
func (*Session) CloseForm()                  {}
func (*Session) EnableCoordinates(_ bool)    {}
func (*Session) EnableInstantRespawn(_ bool) {}
func (*Session) HandleInventories(_ *world.Tx, _ session.Controllable, _ *inventory.Inventory, _ *inventory.Inventory, _ *inventory.Inventory, _ *inventory.Inventory, _ *inventory.Armour, _ *uint32) {
}
func (*Session) HideHudElement(_ hud.Element)               {}
func (*Session) HudElementHidden(_ hud.Element) bool        { return false }
func (*Session) InputLocked(_ input.Lock) bool              { return false }
func (*Session) LockInput(_ input.Lock)                     {}
func (*Session) OpenBlockContainer(_ cube.Pos, _ *world.Tx) {}
func (*Session) OpenSign(_ cube.Pos, _ bool)                {}
func (*Session) OpenTrade(_ *world.Tx, _ world.Entity, _ string, _ []session.TradeOffer, _ func(index int, times int) bool) {
}
func (*Session) PlaySound(_ world.Sound, _ mgl64.Vec3)                               {}
func (*Session) RemoveAllDebugShapes()                                               {}
func (*Session) RemoveBossBar()                                                      {}
func (*Session) RemoveDebugShape(_ debug.Shape)                                      {}
func (*Session) RemoveScoreboard()                                                   {}
func (*Session) RemoveViewLayer(_ world.Entity)                                      {}
func (*Session) SendAbilities(_ session.Controllable)                                {}
func (*Session) SendActionBarMessage(_ string)                                       {}
func (*Session) SendBossBar(_ string, _ uint8, _ float64)                            {}
func (*Session) SendChargeItemComplete()                                             {}
func (*Session) SendCommandOutput(_ *cmd.Output, _ language.Tag)                     {}
func (*Session) SendDebugShapes(_ world.Dimension)                                   {}
func (*Session) SendDialogue(_ dialogue.Dialogue, _ world.Entity)                    {}
func (*Session) SendEffect(_ effect.Effect)                                          {}
func (*Session) SendEffectRemoval(_ effect.Type)                                     {}
func (*Session) SendExperience(_ int, _ float64)                                     {}
func (*Session) SendFood(_ int, _ float64, _ float64)                                {}
func (*Session) SendForm(_ form.Form)                                                {}
func (*Session) SendGameMode(_ session.Controllable)                                 {}
func (*Session) SendHealth(_ float64, _ float64, _ float64)                          {}
func (*Session) SendHeldSlot(_ int, _ session.Controllable, _ bool)                  {}
func (*Session) SendHudUpdates()                                                     {}
func (*Session) SendInputLocks()                                                     {}
func (*Session) SendJukeboxPopup(_ string)                                           {}
func (*Session) SendMessage(_ string)                                                {}
func (*Session) SendPlayerSpawn(_ mgl64.Vec3)                                        {}
func (*Session) SendPopup(_ string)                                                  {}
func (*Session) SendRespawn(_ mgl64.Vec3, _ session.Controllable)                    {}
func (*Session) SendScoreboard(_ *scoreboard.Scoreboard)                             {}
func (*Session) SendSpeed(_ float64)                                                 {}
func (*Session) SendSubtitle(_ string)                                               {}
func (*Session) SendTip(_ string)                                                    {}
func (*Session) SendTitle(_ string)                                                  {}
func (*Session) SendToast(_ string, _ string)                                        {}
func (*Session) SendTranslation(_ chat.Translation, _ language.Tag, _ []any)         {}
func (*Session) SetHandle(_ *world.EntityHandle, _ skin.Skin)                        {}
func (*Session) SetTitleDurations(_ time.Duration, _ time.Duration, _ time.Duration) {}
func (*Session) ShowHudElement(_ hud.Element)                                        {}
func (*Session) StartShowingEntity(_ world.Entity)                                   {}
func (*Session) StopShowingEntity(_ world.Entity)                                    {}
func (*Session) Transfer(_ net.IP, _ int)                                            {}
func (*Session) UnlockInput(_ input.Lock)                                            {}
func (*Session) UpdateTradeOffers(_ []session.TradeOffer)                            {}
func (*Session) ViewAlwaysShowNameTag(_ world.Entity, _ bool)                        {}
func (*Session) ViewBlockAction(_ cube.Pos, _ world.BlockAction)                     {}
func (*Session) ViewBrewingUpdate(_ time.Duration, _ time.Duration, _ int32, _ int32, _ int32, _ int32) {
}
func (*Session) ViewEmote(_ world.Entity, _ uuid.UUID)                       {}
func (*Session) ViewEntityAnimation(_ world.Entity, _ world.EntityAnimation) {}
func (*Session) ViewEntityArmour(_ world.Entity)                             {}
func (*Session) ViewEntityDismount(_ world.Entity, _ world.Entity)           {}
func (*Session) ViewEntityGameMode(_ world.Entity)                           {}
func (*Session) ViewEntityItems(_ world.Entity)                              {}
func (*Session) ViewEntityMount(_ world.Entity, _ world.Entity, _ bool)      {}
func (*Session) ViewEntityState(_ world.Entity)                              {}
func (*Session) ViewEntityWake(_ world.Entity)                               {}
func (*Session) ViewFurnaceUpdate(_ time.Duration, _ time.Duration, _ time.Duration, _ time.Duration, _ time.Duration, _ time.Duration) {
}
func (*Session) ViewItemCooldown(_ world.Item, _ time.Duration)         {}
func (*Session) ViewLayer() *world.ViewLayer                            { return nil }
func (*Session) ViewNameTag(_ world.Entity, _ string)                   {}
func (*Session) ViewParticle(_ mgl64.Vec3, _ world.Particle)            {}
func (*Session) ViewPublicAlwaysShowNameTag(_ world.Entity)             {}
func (*Session) ViewPublicNameTag(_ world.Entity)                       {}
func (*Session) ViewPublicScoreTag(_ world.Entity)                      {}
func (*Session) ViewScoreTag(_ world.Entity, _ string)                  {}
func (*Session) ViewSkin(_ world.Entity)                                {}
func (*Session) ViewSleepingPlayers(_ int, _ int)                       {}
func (*Session) ViewSound(_ mgl64.Vec3, _ world.Sound)                  {}
func (*Session) ViewTime(_ int)                                         {}
func (*Session) ViewTimeCycle(_ bool)                                   {}
func (*Session) ViewVisibility(_ world.Entity, _ world.VisibilityLevel) {}
func (*Session) ViewWeather(_ bool, _ bool)                             {}
func (*Session) ViewWorldSpawn(_ cube.Pos)                              {}
func (*Session) VisibleDebugShapes() []debug.Shape                      { return nil }
