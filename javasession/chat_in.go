package javasession

import (
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	v777 "github.com/ezchr/go-mc/java/v777"
	"github.com/ezchr/go-mc/java/wire"
)

// handleTextPacket handles the client's chat, command, dialog and sign packets. It reports
// whether id was one of them. Signatures and acknowledgements are ignored: the server is offline
// mode and login says enforces_secure_chat=false.
func (s *Session) handleTextPacket(id int32, body []byte) (handled bool, err error) {
	r := wire.NewReader(body)
	switch id {
	case v777.ServerboundPlayChat:
		msg := r.String(256)
		// timestamp, salt, optional signature, acknowledgements: ignored.
		if r.Err != nil {
			return true, r.Err
		}
		if !validChat(msg) {
			s.log.Debug("chat with illegal characters dropped")
			return true, nil
		}
		s.inTx(func(c session.Controllable) { c.Chat(msg) })
	case v777.ServerboundPlayChatCommand, v777.ServerboundPlayChatCommandSigned:
		command := r.String(32767)
		if r.Err != nil {
			return true, r.Err
		}
		if !validChat(command) {
			return true, nil
		}
		s.inTx(func(c session.Controllable) { c.ExecuteCommand("/" + command) })
	case v777.ServerboundPlayChatAck, v777.ServerboundPlayChatSessionUpdate, v777.ServerboundPlayCommandSuggestion:
		// Nothing to do: no signed chat, no server-side suggestions yet.
	case v777.ServerboundPlayCustomClickAction:
		ident := r.String(32767)
		payload := r.ByteArray(65536)
		if r.Err != nil {
			return true, r.Err
		}
		s.handleClickAction(ident, payload)
	case v777.ServerboundPlaySignUpdate:
		x, y, z := r.Position()
		var lines [4]string
		for i := range lines {
			lines[i] = r.String(384)
		}
		front := r.VarInt() == 1 // SignTextSlot: BACK 0, FRONT 1
		if r.Err != nil {
			return true, r.Err
		}
		s.editSign(cube.Pos{x, y, z}, front, lines)
	default:
		return false, nil
	}
	return true, nil
}

// validChat rejects what vanilla rejects in chat (section signs, control characters, DEL).
func validChat(s string) bool {
	for _, r := range s {
		if r == '§' || r < ' ' || r == 0x7f {
			return false
		}
	}
	return true
}

// inTx runs f in the player's transaction, logging failures other than the player being gone.
func (s *Session) inTx(f func(c session.Controllable)) {
	err := s.withPlayer(func(_ *world.Tx, c session.Controllable) { f(c) })
	if err != nil && !stopped(err) {
		s.log.Debug("text packet", "err", err)
	}
}

// OpenSign opens the sign editor for one side of the sign at pos.
func (s *Session) OpenSign(pos cube.Pos, frontSide bool) {
	w := s.packet()
	w.Position(pos[0], pos[1], pos[2])
	if frontSide {
		w.VarInt(1)
	} else {
		w.VarInt(0)
	}
	s.queue(v777.ClientboundPlayOpenSignEditor, w)
}

// editSign applies the four lines a Java client wrote on one side of a sign. The other side keeps
// its text.
func (s *Session) editSign(pos cube.Pos, front bool, lines [4]string) {
	n := len(lines)
	for n > 0 && lines[n-1] == "" {
		n--
	}
	t := strings.Join(lines[:n], "\n")
	err := s.withPlayer(func(tx *world.Tx, c session.Controllable) {
		sign, ok := tx.Block(pos).(block.Sign)
		if !ok {
			return
		}
		frontText, backText := sign.Front.Text, sign.Back.Text
		if front {
			frontText = t
		} else {
			backText = t
		}
		if err := c.EditSign(pos, frontText, backText); err != nil {
			s.log.Debug("edit sign", "err", err)
		}
	})
	if err != nil && !stopped(err) {
		s.log.Debug("edit sign", "err", err)
	}
}
