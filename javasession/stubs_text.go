package javasession

// Not implemented yet (text area): these do nothing. Generated with tools/stubgen from
// player.Session; delete a method here when it gets a real implementation.

import (
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player/chat"
	"github.com/df-mc/dragonfly/server/player/dialogue"
	"github.com/df-mc/dragonfly/server/player/form"
	"github.com/df-mc/dragonfly/server/player/scoreboard"
	"github.com/df-mc/dragonfly/server/world"
	"golang.org/x/text/language"
)

// Methods the Java session does not implement yet: they do nothing. Generated with
// tools/stubgen from player.Session; delete a line when the method gets a real implementation.

func (*Session) CloseDialogue()                                                      {}
func (*Session) CloseForm()                                                          {}
func (*Session) OpenSign(_ cube.Pos, _ bool)                                         {}
func (*Session) RemoveBossBar()                                                      {}
func (*Session) RemoveScoreboard()                                                   {}
func (*Session) SendActionBarMessage(_ string)                                       {}
func (*Session) SendBossBar(_ string, _ uint8, _ float64)                            {}
func (*Session) SendCommandOutput(_ *cmd.Output, _ language.Tag)                     {}
func (*Session) SendDialogue(_ dialogue.Dialogue, _ world.Entity)                    {}
func (*Session) SendForm(_ form.Form)                                                {}
func (*Session) SendJukeboxPopup(_ string)                                           {}
func (*Session) SendMessage(_ string)                                                {}
func (*Session) SendPopup(_ string)                                                  {}
func (*Session) SendScoreboard(_ *scoreboard.Scoreboard)                             {}
func (*Session) SendSubtitle(_ string)                                               {}
func (*Session) SendTip(_ string)                                                    {}
func (*Session) SendTitle(_ string)                                                  {}
func (*Session) SendToast(_ string, _ string)                                        {}
func (*Session) SendTranslation(_ chat.Translation, _ language.Tag, _ []any)         {}
func (*Session) SetTitleDurations(_ time.Duration, _ time.Duration, _ time.Duration) {}
func (*Session) ViewAlwaysShowNameTag(_ world.Entity, _ bool)                        {}
func (*Session) ViewNameTag(_ world.Entity, _ string)                                {}
func (*Session) ViewPublicAlwaysShowNameTag(_ world.Entity)                          {}
func (*Session) ViewPublicNameTag(_ world.Entity)                                    {}
func (*Session) ViewPublicScoreTag(_ world.Entity)                                   {}
func (*Session) ViewScoreTag(_ world.Entity, _ string)                               {}
