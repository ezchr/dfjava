package javasession

import "sync"

// itemStates holds each session's inventory state until Session has a field for it.
var itemStates sync.Map // *Session -> *itemState

// items returns the session's inventory state.
//
// TODO: once Session has the field `items *itemState` (set in newSession with
// `s.items = newItemState(s)`), this becomes `return s.items` and itemStates goes.
func (s *Session) items() *itemState {
	if v, ok := itemStates.Load(s); ok {
		return v.(*itemState)
	}
	st := newItemState(s)
	if v, loaded := itemStates.LoadOrStore(s, st); loaded {
		return v.(*itemState)
	}
	go func() {
		<-s.closed
		itemStates.Delete(s)
	}()
	return st
}
