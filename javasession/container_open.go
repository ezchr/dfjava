package javasession

import (
	"reflect"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	v777 "github.com/ezchr/go-mc/java/v777"
)

// menu returns the open window (the player's inventory if no block window is open).
func (st *itemState) menu() *menu {
	if m := st.open.Load(); m != nil {
		return m
	}
	return playerMenu
}

// nextWindowID is vanilla's container counter: 1-100, then around again.
func (st *itemState) nextWindowID() int32 {
	st.windowID = st.windowID%100 + 1
	return st.windowID
}

// OpenBlockContainer opens the window of the block at pos (chests, furnaces, crafting tables...).
func (s *Session) OpenBlockContainer(pos cube.Pos, tx *world.Tx) {
	st := s.items()
	if m := st.open.Load(); m != nil && m.pos == pos {
		return
	}
	_, c := st.current()
	if c == nil || st.inv == nil {
		return
	}
	if _, _, ok := menuFor(tx.Block(pos)); !ok {
		return
	}
	s.closeMenu(tx, c, true)

	b := tx.Block(pos)
	switch cb := b.(type) {
	case block.EnderChest:
		cb.AddViewer(tx, pos)
	case block.Container:
		cb.AddViewer(s, tx, pos) // a chest may pair with its neighbour here
		b = tx.Block(pos)
	}
	m, title, ok := menuFor(b)
	if !ok {
		return
	}
	m.pos, m.w = pos, tx.World()
	switch {
	case m.ender:
		m.inv = c.EnderChestInventory()
	case m.kind == menuChest || m.kind == menuShulker || m.kind == menuHopper || m.kind == menuFurnace || m.kind == menuBrewing:
		m.inv = b.(block.Container).Inventory(tx, pos)
		if m.kind == menuChest && m.inv.Size() != m.size {
			m.size = m.inv.Size()
		}
	}
	m.id = st.nextWindowID()
	st.drag = dragState{}
	st.open.Store(m)

	w := s.packet()
	w.VarInt(m.id)
	w.VarInt(m.typ)
	title.Write(w)
	s.queue(v777.ClientboundPlayOpenScreen, w)
	s.syncWindow(tx, c, m)
}

// closeMenu closes the open block window (if any): the block forgets the viewer and the items in
// the window's input slots and on the cursor go back to the inventory. send tells the client
// (it is false when the client closed the window itself).
func (s *Session) closeMenu(tx *world.Tx, c session.Controllable, send bool) {
	st := s.items()
	m := st.open.Swap(nil)
	if m == nil {
		return
	}
	st.drag = dragState{}
	if send {
		w := s.packet()
		w.VarInt(m.id)
		s.queue(v777.ClientboundPlayContainerClose, w)
	}
	var b world.Block
	if tx.World() == m.w {
		b = tx.Block(m.pos)
	}
	switch b := b.(type) {
	case block.EnderChest:
		if m.ender {
			b.RemoveViewer(tx, m.pos)
		}
	case block.Container:
		b.RemoveViewer(s, tx, m.pos)
	}
	st.applying.Store(true)
	c.MoveItemsToInventory()
	st.applying.Store(false)
	s.sendInventory()
}

// clientClosed handles container_close. Like vanilla, the window id hardly matters: whatever is
// open closes.
func (s *Session) clientClosed(tx *world.Tx, c session.Controllable, window int32) {
	st := s.items()
	if st.open.Load() != nil {
		s.closeMenu(tx, c, false)
		return
	}
	st.drag = dragState{}
	if window == windowPlayer && st.inv != nil {
		// Vanilla puts the cursor and the crafting grid back into the inventory, or drops them.
		st.applying.Store(true)
		c.MoveItemsToInventory()
		st.applying.Store(false)
		s.sendInventory()
	}
}

// checkMenu closes the open window when the player walked away from the block or the block is gone
// (vanilla stillValid). It is called from HandleInventories, so at most every few ticks it looks.
func (s *Session) checkMenu(tx *world.Tx, c session.Controllable) {
	st := s.items()
	m := st.open.Load()
	if m == nil {
		return
	}
	now := time.Now()
	if now.Sub(m.lastCheck) < 200*time.Millisecond {
		return
	}
	m.lastCheck = now
	if tx.World() == m.w && reflect.TypeOf(tx.Block(m.pos)) == m.btype && c.Position().Sub(m.pos.Vec3Centre()).Len() <= 8 {
		return
	}
	// HandleInventories runs while the player entity is being opened: close in a transaction of its own.
	go s.do(func(tx *world.Tx, c session.Controllable) {
		if st.open.Load() == m {
			s.closeMenu(tx, c, true)
		}
	})
}

// syncWindow sends the whole window (and its data) again: the click engine resyncs after every
// click instead of trusting the client's prediction.
func (s *Session) syncWindow(tx *world.Tx, c session.Controllable, m *menu) {
	if m.kind == menuPlayer {
		s.sendInventory()
		return
	}
	st := s.items()
	if st.inv == nil {
		return
	}
	v := st.newView(tx, c, m)
	w := s.packet()
	w.VarInt(m.id)
	w.VarInt(st.nextStateID())
	w.VarInt(int32(v.n))
	for js := range v.n {
		s.writeStack(w, v.slots[js])
	}
	s.writeStack(w, v.cursor)
	s.queue(v777.ClientboundPlayContainerSetContent, w)
	s.syncMenuData(tx, c, m, v)
}

// syncMenuData sends the window's data slots (progress bars, costs) that changed, all of them the
// first time like vanilla's initMenu.
func (s *Session) syncMenuData(tx *world.Tx, c session.Controllable, m *menu, v *view) {
	var vals [10]int32
	n := 0
	switch m.kind {
	case menuFurnace:
		n = 4
		if d, ok := tx.Block(m.pos).(interface {
			Durations() (remaining, max, cook time.Duration)
		}); ok {
			rem, mx, cook := d.Durations()
			vals = [10]int32{durTicks(rem), durTicks(mx), durTicks(cook), m.cookTotal}
		}
	case menuBrewing:
		n = 4
		if b, ok := tx.Block(m.pos).(interface {
			Duration() time.Duration
			Fuel() (int32, int32)
		}); ok {
			fuel, total := b.Fuel()
			vals = [10]int32{durTicks(b.Duration()), fuel, 400, total}
		}
	case menuAnvil:
		n, vals[0] = 1, int32(m.cost)
	case menuSmithing:
		n = 1 // has recipe error: never
	case menuStonecutter:
		n, vals[0] = 1, int32(m.sel)
	case menuEnchanting:
		n, vals = 10, enchantData(tx, c, m.pos, v.slots[0])
	case menuBeacon:
		n = 3
		if b, ok := tx.Block(m.pos).(block.Beacon); ok {
			vals[0], vals[1], vals[2] = int32(b.Level()), beaconEffectData(b.Primary), beaconEffectData(b.Secondary)
		}
	}
	for i := range n {
		if m.dataSent && m.data[i] == vals[i] {
			continue
		}
		m.data[i] = vals[i]
		s.sendData(m.id, i, vals[i])
	}
	m.dataN, m.dataSent = n, true
}

func (s *Session) sendData(window int32, key int, val int32) {
	w := s.packet()
	w.VarInt(window)
	w.Int16(int16(key))
	w.Int16(int16(val))
	s.queue(v777.ClientboundPlayContainerSetData, w)
}

func durTicks(d time.Duration) int32 { return int32(d / (50 * time.Millisecond)) }

// ViewSlotChange shows a change in the open block's inventory (another player, a hopper, smelting).
func (s *Session) ViewSlotChange(slot int, it item.Stack) {
	st := s.items()
	m := st.open.Load()
	if m == nil || m.inv == nil || m.ender || st.applying.Load() {
		return
	}
	js := slot
	if m.kind == menuBrewing {
		js = brewingWindowSlot(slot)
	}
	if js >= m.size {
		return
	}
	s.sendWindowSlot(m.id, js, it)
}

func (s *Session) sendWindowSlot(window int32, js int, it item.Stack) {
	w := s.packet()
	w.VarInt(window)
	w.VarInt(s.items().nextStateID())
	w.Int16(int16(js))
	s.writeStack(w, it)
	s.queue(v777.ClientboundPlayContainerSetSlot, w)
}

// ViewFurnaceUpdate updates the progress arrow and flame of the open furnace.
func (s *Session) ViewFurnaceUpdate(prevCook, cook, prevRemaining, remaining, prevMax, max time.Duration) {
	m := s.items().open.Load()
	if m == nil || m.kind != menuFurnace {
		return
	}
	s.updateData(m, 0, prevRemaining != remaining, durTicks(remaining))
	s.updateData(m, 1, prevMax != max, durTicks(max))
	s.updateData(m, 2, prevCook != cook, durTicks(cook))
}

// ViewBrewingUpdate updates the bubbles and fuel bar of the open brewing stand.
func (s *Session) ViewBrewingUpdate(prevBrew, brew time.Duration, prevFuel, fuel, prevTotal, total int32) {
	m := s.items().open.Load()
	if m == nil || m.kind != menuBrewing {
		return
	}
	s.updateData(m, 0, prevBrew != brew, durTicks(brew))
	s.updateData(m, 1, prevFuel != fuel, fuel)
	s.updateData(m, 3, prevTotal != total, total)
}

func (s *Session) updateData(m *menu, key int, changed bool, val int32) {
	if !changed || (m.dataSent && m.data[key] == val) {
		return
	}
	m.data[key] = val
	s.sendData(m.id, key, val)
}

// enderSlotChanged shows a change in the ender chest while it is open (a slot func of the ender
// chest inventory).
func (s *Session) enderSlotChanged(slot int, it item.Stack) {
	st := s.items()
	m := st.open.Load()
	if m == nil || !m.ender || st.applying.Load() {
		return
	}
	s.sendWindowSlot(m.id, slot, it)
}

// uiSlotChanged shows a change in the UI inventory made outside a click (plugins, closing).
func (s *Session) uiSlotChanged(slot int, it item.Stack) {
	st := s.items()
	if st.applying.Load() {
		return
	}
	m := st.menu()
	if m.kind == menuPlayer {
		if slot >= uiCraftSmall && slot < uiCraftSmall+4 {
			s.sendSlot(1+slot-uiCraftSmall, it)
			s.sendSlot(0, st.playerCraftResult())
		}
		return
	}
	for js, sec := range uiSection[m.kind] {
		if sec.ui == slot {
			s.sendWindowSlot(m.id, js, it)
		}
	}
}

// closeContainers is for Session.Close, before the player is saved: it closes the open window
// without telling the client (the block forgets the viewer) and puts the items of the UI inventory
// (crafting grid, station inputs, cursor) back into the inventory, as the Bedrock session does.
func (s *Session) closeContainers(tx *world.Tx, c session.Controllable) {
	st := s.items()
	if st.inv == nil || tx == nil || c == nil {
		return
	}
	if st.open.Load() != nil {
		s.closeMenu(tx, c, false)
		return
	}
	st.applying.Store(true)
	c.MoveItemsToInventory()
	st.applying.Store(false)
}
