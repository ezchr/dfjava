package javasession

import (
	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	jitem "github.com/ezchr/go-mc/java/item"
	v777 "github.com/ezchr/go-mc/java/v777"
	"github.com/ezchr/go-mc/java/wire"
)

// container_click modes (ContainerInput).
const (
	clickPickup = iota
	clickQuickMove
	clickSwap
	clickClone
	clickThrow
	clickQuickCraft
	clickPickupAll
)

// handleInventoryPacket handles the inventory packets: held slot, creative slots, clicks in and
// closing of the player's inventory. handled is false for other packets.
func (s *Session) handleInventoryPacket(id int32, body []byte) (handled bool, err error) {
	r := wire.NewReader(body)
	st := s.items()
	switch id {
	case v777.ServerboundPlaySetCarriedItem:
		slot := int(r.Int16())
		if r.Err != nil {
			return true, r.Err
		}
		if slot < 0 || slot > 8 {
			return true, nil // vanilla ignores it too
		}
		s.do(func(_ *world.Tx, c session.Controllable) {
			st.changingSlot.Store(true)
			defer st.changingSlot.Store(false)
			_ = c.SetHeldSlot(slot)
		})
	case v777.ServerboundPlaySetCreativeModeSlot:
		slot := int(r.Int16())
		st.in.DecodeUntrusted(r)
		if r.Err != nil {
			return true, r.Err
		}
		s.do(func(_ *world.Tx, c session.Controllable) { s.creativeSlot(c, slot, &st.in) })
	case v777.ServerboundPlayContainerClick:
		var k click
		window := r.VarInt()
		r.VarInt() // state id: we resync after every click anyway
		k.slot = int(r.Int16())
		k.button = int(r.Int8())
		k.mode = int(r.VarInt())
		n := int(r.VarInt())
		if r.Err == nil && (n < 0 || n > 128) {
			return true, jitem.ErrInvalid
		}
		var hs jitem.HashedStack
		for range n {
			r.Int16()
			hs.Decode(r)
		}
		hs.Decode(r) // carried
		if r.Err != nil {
			return true, r.Err
		}
		s.do(func(_ *world.Tx, c session.Controllable) {
			if window == windowPlayer {
				s.click(c, k)
			}
			s.sendInventory()
		})
	case v777.ServerboundPlayContainerClose:
		window := r.VarInt()
		if r.Err != nil {
			return true, r.Err
		}
		if window == windowPlayer {
			st.drag = dragState{}
			// Vanilla puts the cursor (and crafting grid) back into the inventory, or drops it.
			s.do(func(_ *world.Tx, c session.Controllable) { c.MoveItemsToInventory() })
		}
	default:
		return false, nil
	}
	return true, nil
}

// dropHeldItem drops the held item (Q, or Ctrl+Q for the whole stack): call it for player_action
// DROP_ITEM / DROP_ALL_ITEMS.
func (s *Session) dropHeldItem(c session.Controllable, all bool) {
	st := s.items()
	if st.inv == nil {
		return
	}
	slot := int(*st.heldSlot)
	it, _ := st.inv.Item(slot)
	if it.Empty() {
		return
	}
	if !all {
		it = it.Grow(1 - it.Count())
	}
	ctx := event.C(inventory.Holder(c))
	if st.inv.Handler().HandleDrop(ctx, slot, it); ctx.Cancelled() {
		s.sendSlot(mainToJava(slot), st.slotItem(mainToJava(slot)))
		return
	}
	n := c.Drop(it)
	cur, _ := st.inv.Item(slot)
	_ = st.inv.SetItem(slot, cur.Grow(-n))
	if n < it.Count() {
		s.sendSlot(mainToJava(slot), st.slotItem(mainToJava(slot)))
	}
}

// swapHands swaps the main hand and off-hand items (F): call it for player_action
// SWAP_ITEM_WITH_OFFHAND.
func (s *Session) swapHands(c session.Controllable) {
	main, off := c.HeldItems()
	c.SetHeldItems(off, main)
}

// creativeSlot handles set_creative_mode_slot: a creative client sets a slot of its inventory (or
// drops a stack, slot -1) to a stack it made.
func (s *Session) creativeSlot(c session.Controllable, slot int, js *jitem.Stack) {
	st := s.items()
	if !c.GameMode().CreativeInventory() || st.inv == nil {
		s.sendInventory()
		return
	}
	ds, ok := st.creativeStack(js)
	if !ok {
		s.sendInventory()
		return
	}
	if slot == -1 {
		if !ds.Empty() {
			c.Drop(ds)
		}
		return
	}
	inv, i, ok := st.slotRef(slot)
	if !ok {
		return
	}
	if !mayPlace(slot, ds) {
		s.sendSlot(slot, st.slotItem(slot))
		return
	}
	if old, _ := inv.Item(i); !old.Empty() {
		copy(st.recent[:], st.recent[1:])
		st.recent[len(st.recent)-1] = old
	}
	_ = inv.SetItem(i, ds)
}

// creativeStack turns a stack from a creative client into a Dragonfly stack. A stack the server
// sent (moved around by the client) is matched to the original, so data Java items can't carry
// (Dragonfly item values) survives; anything else is converted.
func (st *itemState) creativeStack(js *jitem.Stack) (item.Stack, bool) {
	if js.Empty() {
		return item.Stack{}, true
	}
	count := js.Count
	js.Count = 1
	var want, got wire.Writer
	js.Encode(&want)
	js.Count = count
	var conv jitem.Stack
	match := func(ds item.Stack) bool {
		if ds.Empty() {
			return false
		}
		got.Reset()
		javaStack(ds, &conv)
		conv.Count = 1
		conv.Encode(&got)
		return string(got.B) == string(want.B)
	}
	try := func(ds item.Stack) (item.Stack, bool) {
		if !match(ds) {
			return item.Stack{}, false
		}
		return ds.Grow(max(1, min(int(count), ds.MaxCount())) - ds.Count()), true
	}
	for js := range playerSlots {
		if ds, ok := try(st.slotItem(js)); ok {
			return ds, true
		}
	}
	for i := len(st.recent) - 1; i >= 0; i-- {
		if ds, ok := try(st.recent[i]); ok {
			return ds, true
		}
	}
	return dragonflyStack(js)
}

// click is a container_click in the player's window.
type click struct {
	slot, button, mode int
}

// dragState is a quick-craft (drag) in progress.
type dragState struct {
	active bool
	kind   int // 0: split evenly, 1: one each, 2: full stacks (creative)
	slots  []int
}

// view is the player window during a click: changes are made here, checked against the inventory
// handlers, then applied at once.
type view struct {
	st     *itemState
	c      session.Controllable
	orig   [playerSlots]item.Stack
	slots  [playerSlots]item.Stack
	cursor item.Stack
	ocur   item.Stack
	drops  []drop
}

// drop is a stack a click throws out of the window, from a slot or the cursor.
type drop struct {
	from int // Java slot, or cursorDrop
	it   item.Stack
}

const cursorDrop = -1

func (s *Session) click(c session.Controllable, k click) {
	st := s.items()
	if st.inv == nil {
		return
	}
	v := &view{st: st, c: c}
	for js := range playerSlots {
		v.orig[js] = st.slotItem(js)
	}
	v.slots = v.orig
	v.ocur = st.cursor()
	v.cursor = v.ocur
	creative := c.GameMode().CreativeInventory()

	if k.mode != clickQuickCraft {
		st.drag = dragState{}
	}
	switch k.mode {
	case clickPickup:
		v.pickup(k.slot, k.button)
	case clickQuickMove:
		if v.valid(k.slot) {
			v.quickMove(k.slot)
		}
	case clickSwap:
		v.swap(k.slot, k.button)
	case clickClone:
		if creative && v.valid(k.slot) && v.cursor.Empty() && !v.slots[k.slot].Empty() {
			it := v.slots[k.slot]
			v.cursor = it.Grow(it.MaxCount() - it.Count())
		}
	case clickThrow:
		if v.valid(k.slot) && v.cursor.Empty() && !v.slots[k.slot].Empty() {
			it := v.slots[k.slot]
			n := it.Count()
			if k.button == 0 {
				n = 1
			}
			v.drops = append(v.drops, drop{k.slot, it.Grow(n - it.Count())})
			v.slots[k.slot] = it.Grow(-n)
		}
	case clickQuickCraft:
		v.quickCraft(k, creative)
	case clickPickupAll:
		v.pickupAll(k.slot)
	}
	v.commit()
}

// valid reports whether js is a slot items can be taken from or put in (not the crafting slots,
// which Dragonfly has no Java crafting for yet).
func (v *view) valid(js int) bool { return js > slotCraftEnd && js < playerSlots }

// mayPlace reports whether stack it can go in Java slot js.
func mayPlace(js int, it item.Stack) bool {
	if js <= slotCraftEnd || js >= playerSlots {
		return false
	}
	if it.Empty() || js < slotArmour || js >= slotMain {
		return true
	}
	switch js - slotArmour {
	case 0:
		h, ok := it.Item().(item.HelmetType)
		return ok && h.Helmet()
	case 1:
		c, ok := it.Item().(item.ChestplateType)
		return ok && c.Chestplate()
	case 2:
		l, ok := it.Item().(item.LeggingsType)
		return ok && l.Leggings()
	default:
		b, ok := it.Item().(item.BootsType)
		return ok && b.Boots()
	}
}

// maxIn is how many of it fit in slot js.
func maxIn(js int, it item.Stack) int {
	if js >= slotArmour && js < slotMain {
		return 1
	}
	return it.MaxCount()
}

func (v *view) pickup(js, button int) {
	if button != 0 && button != 1 {
		return
	}
	if js == slotOutside {
		if !v.cursor.Empty() {
			n := v.cursor.Count()
			if button == 1 {
				n = 1
			}
			v.drops = append(v.drops, drop{cursorDrop, v.cursor.Grow(n - v.cursor.Count())})
			v.cursor = v.cursor.Grow(-n)
		}
		return
	}
	if !v.valid(js) {
		return
	}
	it, cur := v.slots[js], v.cursor
	switch {
	case it.Empty():
		if cur.Empty() || !mayPlace(js, cur) {
			return
		}
		n := cur.Count()
		if button == 1 {
			n = 1
		}
		n = min(n, maxIn(js, cur))
		v.slots[js] = cur.Grow(n - cur.Count())
		v.cursor = cur.Grow(-n)
	case cur.Empty():
		n := it.Count()
		if button == 1 {
			n = (n + 1) / 2
		}
		v.cursor = it.Grow(n - it.Count())
		v.slots[js] = it.Grow(-n)
	case mayPlace(js, cur) && it.Comparable(cur):
		n := cur.Count()
		if button == 1 {
			n = 1
		}
		n = min(n, maxIn(js, it)-it.Count())
		if n > 0 {
			v.slots[js] = it.Grow(n)
			v.cursor = cur.Grow(-n)
		}
	case mayPlace(js, cur) && cur.Count() <= maxIn(js, cur):
		v.slots[js], v.cursor = cur, it
	}
}

// moveTo moves as much of it as fits into slots [from, to): first onto equal stacks, then into empty
// slots (AbstractContainerMenu.moveItemStackTo). It returns what is left.
func (v *view) moveTo(it item.Stack, from, to int) item.Stack {
	for js := from; js < to && !it.Empty(); js++ {
		s := v.slots[js]
		if s.Empty() || !s.Comparable(it) || !mayPlace(js, it) {
			continue
		}
		n := min(it.Count(), maxIn(js, s)-s.Count())
		if n > 0 {
			v.slots[js] = s.Grow(n)
			it = it.Grow(-n)
		}
	}
	for js := from; js < to && !it.Empty(); js++ {
		if !v.slots[js].Empty() || !mayPlace(js, it) {
			continue
		}
		n := min(it.Count(), maxIn(js, it))
		v.slots[js] = it.Grow(n - it.Count())
		it = it.Grow(-n)
	}
	return it
}

// quickMove is a shift-click (InventoryMenu.quickMoveStack).
func (v *view) quickMove(js int) {
	it := v.slots[js]
	if it.Empty() {
		return
	}
	var left item.Stack
	armourSlot := -1
	for a := slotArmour; a < slotMain; a++ {
		if mayPlace(a, it) {
			armourSlot = a
			break
		}
	}
	switch {
	case js >= slotArmour && js < slotMain, js == slotOffhand:
		left = v.moveTo(it, slotMain, slotOffhand)
	case armourSlot >= 0 && v.slots[armourSlot].Empty():
		left = v.moveTo(it, armourSlot, armourSlot+1)
	case js >= slotMain && js < slotHotbar:
		left = v.moveTo(it, slotHotbar, slotOffhand)
	default: // hotbar
		left = v.moveTo(it, slotMain, slotHotbar)
	}
	v.slots[js] = left
}

// swap swaps a slot with a hotbar slot (number keys, button 0-8) or the off-hand (F, button 40).
func (v *view) swap(js, button int) {
	var other int
	switch {
	case button >= 0 && button < 9:
		other = slotHotbar + button
	case button == 40:
		other = slotOffhand
	default:
		return
	}
	if !v.valid(js) || js == other {
		return
	}
	a, b := v.slots[js], v.slots[other]
	if !mayPlace(js, b) || b.Count() > maxIn(js, b) {
		return
	}
	v.slots[js], v.slots[other] = b, a
}

// quickCraft handles a drag: start (slot -999), one packet per slot, end (slot -999).
func (v *view) quickCraft(k click, creative bool) {
	d := &v.st.drag
	stage, kind := k.button&3, k.button>>2&3
	switch stage {
	case 0:
		*d = dragState{}
		if v.cursor.Empty() || kind > 2 || (kind == 2 && !creative) {
			return
		}
		d.active, d.kind, d.slots = true, kind, d.slots[:0]
	case 1:
		if !d.active || kind != d.kind || !v.valid(k.slot) || len(d.slots) >= playerSlots {
			return
		}
		it := v.slots[k.slot]
		if !mayPlace(k.slot, v.cursor) || (!it.Empty() && !it.Comparable(v.cursor)) {
			return
		}
		for _, js := range d.slots {
			if js == k.slot {
				return
			}
		}
		if d.kind != 2 && len(d.slots) >= v.cursor.Count() {
			return
		}
		d.slots = append(d.slots, k.slot)
	case 2:
		slots, kind := d.slots, d.kind
		active := d.active
		*d = dragState{slots: slots[:0]}
		if !active || len(slots) == 0 || v.cursor.Empty() {
			return
		}
		if len(slots) == 1 {
			if kind < 2 { // vanilla: a one-slot drag is a click with that button
				v.pickup(slots[0], kind)
			}
			return
		}
		cur := v.cursor
		left := cur.Count()
		for _, js := range slots {
			it := v.slots[js]
			if !mayPlace(js, cur) || (!it.Empty() && !it.Comparable(cur)) {
				continue
			}
			var n int
			switch kind {
			case 0:
				n = cur.Count() / len(slots)
			case 1:
				n = 1
			case 2:
				n = cur.MaxCount()
			}
			have := 0
			if !it.Empty() {
				have = it.Count()
			}
			n = min(n, maxIn(js, cur)-have)
			if kind != 2 {
				n = min(n, left)
			}
			if n <= 0 {
				continue
			}
			v.slots[js] = cur.Grow(have + n - cur.Count())
			if kind != 2 {
				left -= n
			}
		}
		if kind != 2 {
			v.cursor = cur.Grow(left - cur.Count())
		}
	}
}

// pickupAll is a double click: gather equal stacks into the cursor, non-full stacks first.
func (v *view) pickupAll(js int) {
	cur := v.cursor
	if cur.Empty() || (v.valid(js) && !v.slots[js].Empty()) {
		return
	}
	limit := cur.MaxCount()
	for pass := 0; pass < 2 && cur.Count() < limit; pass++ {
		for s := slotArmour; s < playerSlots && cur.Count() < limit; s++ {
			it := v.slots[s]
			if it.Empty() || !it.Comparable(cur) || (pass == 0 && it.Count() == it.MaxCount()) {
				continue
			}
			n := min(it.Count(), limit-cur.Count())
			cur = cur.Grow(n)
			v.slots[s] = it.Grow(-n)
		}
	}
	v.cursor = cur
}

// commit runs the inventory handlers for every change and applies the click if none cancels it.
// The caller resends the whole window either way.
func (v *view) commit() {
	st := v.st
	ctx := event.C(inventory.Holder(v.c))
	changed := false
	for js := range playerSlots {
		before, after := v.orig[js], v.slots[js]
		if before.Equal(after) {
			continue
		}
		changed = true
		inv, i, _ := st.slotRef(js)
		takeAndPlace(ctx, inv.Handler(), i, before, after)
	}
	if !v.ocur.Equal(v.cursor) {
		changed = true
		takeAndPlace(ctx, st.ui.Handler(), cursorUISlot, v.ocur, v.cursor)
	}
	for _, d := range v.drops {
		if d.from == cursorDrop {
			st.ui.Handler().HandleDrop(ctx, cursorUISlot, d.it)
		} else {
			inv, i, _ := st.slotRef(d.from)
			inv.Handler().HandleDrop(ctx, i, d.it)
		}
	}
	if !changed || ctx.Cancelled() {
		return
	}
	st.applying.Store(true)
	for js := range playerSlots {
		if !v.orig[js].Equal(v.slots[js]) {
			inv, i, _ := st.slotRef(js)
			_ = inv.SetItem(i, v.slots[js])
		}
	}
	if !v.ocur.Equal(v.cursor) {
		_ = st.ui.SetItem(cursorUISlot, v.cursor)
	}
	st.applying.Store(false)
	for _, d := range v.drops {
		if n := v.c.Drop(d.it); n < d.it.Count() {
			// A drop a player handler cancelled goes back into the inventory.
			_, _ = st.inv.AddItem(d.it.Grow(-n))
		}
	}
}

// takeAndPlace calls HandleTake and HandlePlace for a slot going from before to after.
func takeAndPlace(ctx *inventory.Context, h inventory.Handler, slot int, before, after item.Stack) {
	same := !before.Empty() && !after.Empty() && before.Comparable(after)
	switch {
	case same && after.Count() < before.Count():
		h.HandleTake(ctx, slot, before.Grow(-after.Count()))
	case same && after.Count() > before.Count():
		h.HandlePlace(ctx, slot, after.Grow(-before.Count()))
	case same:
	default:
		if !before.Empty() {
			h.HandleTake(ctx, slot, before)
		}
		if !after.Empty() {
			h.HandlePlace(ctx, slot, after)
		}
	}
}
