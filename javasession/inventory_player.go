package javasession

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/ezchr/dfjava/javamap"
	jitem "github.com/ezchr/go-mc/java/item"
	v777 "github.com/ezchr/go-mc/java/v777"
	"github.com/ezchr/go-mc/java/wire"
)

// Java player inventory window (container id 0) layout.
const (
	windowPlayer   = 0
	slotCraftEnd   = 4 // 0 result, 1-4 crafting grid
	slotArmour     = 5 // 5-8: head, chest, legs, feet
	slotMain       = 9 // 9-35: Dragonfly inventory slots 9-35
	slotHotbar     = 36
	slotOffhand    = 45
	playerSlots    = 46
	slotOutside    = -999
	cursorUISlot   = 0 // Dragonfly keeps the cursor item in slot 0 of the UI inventory
	maxStateID     = 32767
	recentCreative = 16
)

// Java equipment slots (set_equipment).
const (
	equipMainHand = iota
	equipOffHand
	equipFeet
	equipLegs
	equipChest
	equipHead
)

// itemState is the session's inventory state.
type itemState struct {
	s *Session

	mu       sync.Mutex // guards the inventory pointers and slot funcs
	inv      *inventory.Inventory
	offHand  *inventory.Inventory
	ui       *inventory.Inventory
	armour   *inventory.Armour
	heldSlot *uint32
	fnInv    inventory.SlotFunc
	fnOff    inventory.SlotFunc
	fnArmour inventory.SlotFunc
	fnUI     inventory.SlotFunc
	ender    *inventory.Inventory
	fnEnder  inventory.SlotFunc

	// The open block window (nil: only the player's inventory) and the last window id used. Both
	// are only touched in the player's world transactions.
	open     atomic.Pointer[menu]
	windowID int32

	// The transaction and player of the latest HandleInventories: the slot funcs run inside it.
	ctxMu sync.Mutex
	tx    *world.Tx
	c     session.Controllable

	sent         atomic.Bool  // the full inventory went out
	stateID      atomic.Int32 // container state id the client echoes in clicks
	applying     atomic.Bool  // a click is being applied: no per-slot packets, a full resync follows
	changingSlot atomic.Bool  // the client changed its held slot: don't echo it

	convMu sync.Mutex
	conv   jitem.Stack // conversion scratch (world goroutine)

	// Input side (read goroutine and the world transactions it runs).
	in     jitem.Stack
	drag   dragState
	recent [recentCreative]item.Stack // stacks creative set_creative_mode_slot replaced, newest last
}

func (st *itemState) nextStateID() int32 {
	for {
		old := st.stateID.Load()
		n := (old + 1) & maxStateID
		if st.stateID.CompareAndSwap(old, n) {
			return n
		}
	}
}

// current returns the transaction and player the slot funcs run in.
func (st *itemState) current() (*world.Tx, session.Controllable) {
	st.ctxMu.Lock()
	defer st.ctxMu.Unlock()
	return st.tx, st.c
}

// HandleInventories is called each time the player entity is opened in a transaction. The first
// call sends the whole inventory, the held slot and the recipe data the client needs for its
// windows; later calls also check that an open block window is still valid.
func (s *Session) HandleInventories(tx *world.Tx, c session.Controllable, inv, offHand, enderChest, ui *inventory.Inventory, armour *inventory.Armour, heldSlot *uint32) {
	st := s.items()
	st.ctxMu.Lock()
	st.tx, st.c = tx, c
	st.ctxMu.Unlock()

	st.mu.Lock()
	if st.inv != inv || st.offHand != offHand || st.ui != ui || st.armour != armour || st.ender != enderChest {
		st.inv, st.offHand, st.ui, st.armour, st.heldSlot, st.ender = inv, offHand, ui, armour, heldSlot, enderChest
		inv.SlotFunc(st.fnInv)
		offHand.SlotFunc(st.fnOff)
		armour.Inventory().SlotFunc(st.fnArmour)
		ui.SlotFunc(st.fnUI)
		if enderChest != nil {
			enderChest.SlotFunc(st.fnEnder)
		}
	}
	st.heldSlot = heldSlot
	st.mu.Unlock()

	if !st.sent.Swap(true) {
		s.sendRecipes()
		s.sendInventory()
		s.sendHeldSlot(int(*heldSlot))
		if containerTest {
			s.containerTestSetup()
		}
		return
	}
	s.checkMenu(tx, c)
}

// newItemState makes the inventory state with its slot funcs (allocated once, not per transaction).
func newItemState(s *Session) *itemState {
	st := &itemState{s: s}
	st.fnInv = func(slot int, _, after item.Stack) {
		if hs := st.heldSlot; hs != nil && slot == int(*hs) {
			st.broadcast(world.Viewer.ViewEntityItems)
		}
		if !st.applying.Load() {
			s.sendSlot(mainToJava(slot), after)
		}
	}
	st.fnOff = func(_ int, _, after item.Stack) {
		st.broadcast(world.Viewer.ViewEntityItems)
		if !st.applying.Load() {
			s.sendSlot(slotOffhand, after)
		}
	}
	st.fnArmour = func(slot int, before, after item.Stack) {
		if !st.applying.Load() {
			s.sendSlot(slotArmour+slot, after)
		}
		if !before.Comparable(after) || before.Empty() != after.Empty() {
			st.broadcast(world.Viewer.ViewEntityArmour)
		}
	}
	st.fnUI = func(slot int, _, after item.Stack) {
		if slot == cursorUISlot {
			if !st.applying.Load() {
				s.sendCursor(after)
			}
			return
		}
		s.uiSlotChanged(slot, after)
	}
	st.fnEnder = func(slot int, _, after item.Stack) { s.enderSlotChanged(slot, after) }
	return st
}

// broadcast shows the player's held items or armour to everyone viewing it. A slot func can in
// theory run outside the transaction it was set in (an inventory changed from another goroutine);
// the stale transaction then panics, which must not take the world down.
func (st *itemState) broadcast(f func(world.Viewer, world.Entity)) {
	tx, c := st.current()
	if tx == nil || c == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			st.s.log.Debug("inventory broadcast outside its transaction", "err", r)
		}
	}()
	for _, v := range tx.Viewers(c.Position()) {
		f(v, c)
	}
}

// mainToJava is the Java window slot of Dragonfly main inventory slot i.
func mainToJava(i int) int {
	if i < 9 {
		return slotHotbar + i
	}
	return i
}

// playerRef is the slot behind slot js of the player's window (window 0). The crafting result has
// no inventory behind it: it is computed from the grid, which lives in the UI inventory.
func (st *itemState) playerRef(js int) (slotRef, bool) {
	switch {
	case js == 0:
		return slotRef{kind: kindResult}, true
	case js >= 1 && js <= slotCraftEnd:
		return slotRef{st.ui, uiCraftSmall + js - 1, kindPlain}, true
	case js >= slotArmour && js < slotMain:
		return slotRef{st.armour.Inventory(), js - slotArmour, kindArmour}, true
	case js >= slotMain && js < slotHotbar:
		return slotRef{st.inv, js, kindPlain}, true
	case js >= slotHotbar && js < slotOffhand:
		return slotRef{st.inv, js - slotHotbar, kindPlain}, true
	case js == slotOffhand:
		return slotRef{st.offHand, 0, kindPlain}, true
	}
	return slotRef{}, false
}

// slotItem is the stack in slot js of the player's window.
func (st *itemState) slotItem(js int) item.Stack {
	if js == 0 {
		return st.playerCraftResult()
	}
	r, ok := st.playerRef(js)
	if !ok || r.inv == nil {
		return item.Stack{}
	}
	it, _ := r.inv.Item(r.idx)
	return it
}

func (st *itemState) cursor() item.Stack {
	it, _ := st.ui.Item(cursorUISlot)
	return it
}

// writeStack writes a Dragonfly stack as a Java slot.
func (s *Session) writeStack(w *wire.Writer, ds item.Stack) {
	if ds.Empty() {
		jitem.WriteEmpty(w)
		return
	}
	st := s.items()
	st.convMu.Lock()
	javaStack(ds, &st.conv)
	st.conv.Encode(w)
	st.convMu.Unlock()
}

// sendInventory sends the whole player window and the cursor (container_set_content).
func (s *Session) sendInventory() {
	st := s.items()
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.inv == nil {
		return
	}
	w := s.packet()
	w.VarInt(windowPlayer)
	w.VarInt(st.nextStateID())
	w.VarInt(playerSlots)
	for js := range playerSlots {
		s.writeStack(w, st.slotItem(js))
	}
	s.writeStack(w, st.cursor())
	s.queue(v777.ClientboundPlayContainerSetContent, w)
}

// resendInventory sends the whole inventory again (after a respawn, or when the client is out of sync).
func (s *Session) resendInventory() { s.sendInventory() }

func (s *Session) sendSlot(js int, ds item.Stack) {
	w := s.packet()
	w.VarInt(windowPlayer)
	w.VarInt(s.items().nextStateID())
	w.Int16(int16(js))
	s.writeStack(w, ds)
	s.queue(v777.ClientboundPlayContainerSetSlot, w)
}

func (s *Session) sendCursor(ds item.Stack) {
	w := s.packet()
	s.writeStack(w, ds)
	s.queue(v777.ClientboundPlaySetCursorItem, w)
}

func (s *Session) sendHeldSlot(slot int) {
	w := s.packet()
	w.VarInt(int32(slot))
	s.queue(v777.ClientboundPlaySetHeldSlot, w)
}

// SendHeldSlot changes the client's selected hotbar slot, unless the client made the change itself.
func (s *Session) SendHeldSlot(slot int, _ session.Controllable, force bool) {
	if s.items().changingSlot.Load() && !force {
		return
	}
	s.sendHeldSlot(slot)
}

// ViewEntityItems shows the items another entity holds.
func (s *Session) ViewEntityItems(e world.Entity) {
	id, ok := s.entityID(e)
	if !ok || id == selfEntityID {
		return
	}
	c, ok := e.(item.Carrier)
	if !ok {
		return
	}
	main, off := c.HeldItems()
	w := s.packet()
	w.VarInt(id)
	w.Byte(equipMainHand | 0x80)
	s.writeStack(w, main)
	w.Byte(equipOffHand)
	s.writeStack(w, off)
	s.queue(v777.ClientboundPlaySetEquipment, w)
}

// ViewEntityArmour shows the armour another entity wears.
func (s *Session) ViewEntityArmour(e world.Entity) {
	id, ok := s.entityID(e)
	if !ok || id == selfEntityID {
		return
	}
	a, ok := e.(interface{ Armour() *inventory.Armour })
	if !ok {
		return
	}
	inv := a.Armour()
	if inv == nil {
		return
	}
	w := s.packet()
	w.VarInt(id)
	w.Byte(equipFeet | 0x80)
	s.writeStack(w, inv.Boots())
	w.Byte(equipLegs | 0x80)
	s.writeStack(w, inv.Leggings())
	w.Byte(equipChest | 0x80)
	s.writeStack(w, inv.Chestplate())
	w.Byte(equipHead)
	s.writeStack(w, inv.Helmet())
	s.queue(v777.ClientboundPlaySetEquipment, w)
}

// ViewItemCooldown shows a cooldown on an item (the Java cooldown group is the item's id).
func (s *Session) ViewItemCooldown(it world.Item, d time.Duration) {
	name := jitem.Name(javamap.Item(it))
	if name == "" {
		return
	}
	w := s.packet()
	w.String(name)
	w.VarInt(int32(d / (time.Second / 20)))
	s.queue(v777.ClientboundPlayCooldown, w)
}
