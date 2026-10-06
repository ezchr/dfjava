package javasession

import (
	"sync"

	"github.com/df-mc/dragonfly/server/player/dialogue"
	"github.com/df-mc/dragonfly/server/player/form"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/ezchr/go-mc/java/text"
	v777 "github.com/ezchr/go-mc/java/v777"
	"github.com/ezchr/go-mc/java/wire"
)

// textState is the per-session state of the text features: what the client shows now, so updates
// send only what changed, and the forms waiting for an answer. It lives in a side table keyed by
// session (texts) so the text code needs no field in Session; it is removed when the session closes.
type textState struct {
	mu sync.Mutex

	// Sidebar scoreboard.
	sbShown bool
	sbName  string
	sbLines []string
	sbDesc  bool

	// Boss bar.
	bossShown  bool
	bossText   string
	bossColour int32
	bossHealth float32

	// Forms and NPC dialogues shown as Java dialogs, by id.
	forms     map[int32]form.Form
	dialogues map[int32]dialogue.Dialogue
	nextForm  int32

	// Name and score tags of other players.
	nameOverride  map[*world.EntityHandle]string
	teams         map[*world.EntityHandle]string // team name -> sent for this entity
	scoreOverride map[*world.EntityHandle]string
	scoreShown    map[*world.EntityHandle]string // owner name the score tag was sent for
	belowName     bool

	// Hash of the last commands tree sent.
	cmdHash uint64
}

var texts sync.Map // *Session -> *textState

// txt returns the session's text state, creating it on first use.
func (s *Session) txt() *textState {
	if v, ok := texts.Load(s); ok {
		return v.(*textState)
	}
	v, loaded := texts.LoadOrStore(s, &textState{})
	if !loaded {
		go func() {
			<-s.closed
			texts.Delete(s)
		}()
	}
	return v.(*textState)
}

// bedrockText converts a Dragonfly (Bedrock-formatted) string to a component.
func bedrockText(s string) text.Component { return text.Legacy(s, text.Bedrock) }

// writeText writes a Dragonfly string as a component.
func writeText(w *wire.Writer, s string) {
	c := bedrockText(s)
	c.Write(w)
}

// sendSystem sends a system_chat message (overlay: above the hotbar instead of in the chat).
func (s *Session) sendSystem(c *text.Component, overlay bool) {
	w := s.packet()
	c.Write(w)
	w.Bool(overlay)
	s.queue(v777.ClientboundPlaySystemChat, w)
}
