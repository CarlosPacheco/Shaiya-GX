package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/itemtype"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// Throwing an item on the ground follows the inventory object of Game.exe
// (0x4f54e0): a bag item released over the world asks "Do you want to throw
// away the item?" (SysMsg 455), or for stacks opens the quantity window, and
// the answer goes out as 0x0206 (bag, slot, amount). Equipped items (bag 0)
// are ignored and must be unequipped first. Items whose ReqOG byte
// (record offset 0x24) is set refuse with SysMsg 454 and the error sound.
type itemDrop struct {
	candidate *[2]uint8
	active    bool
	stack     bool
	source    [2]uint8
	item      proto.InventoryItem
	confirm   *ui.Message
}

// noteItemDrop remembers a bag item released outside its window; worldControls
// decides after every panel had its say whether the release was over the world.
func (a *App) noteItemDrop(key [2]uint8) {
	if key[0] == 0 {
		return
	}
	a.drop.candidate = &key
}

func (a *App) dropReleased(in *platform.Input, w, h int, captured bool) {
	d := &a.drop
	if d.candidate == nil {
		return
	}
	key := *d.candidate
	d.candidate = nil
	if key[0] == 0 || captured || a.wv == nil || a.barCovered(in, w, h) || a.emblemCovers(in, w, h) || a.actionBar.SlotAt(in.MouseX, in.MouseY, w, h) >= 0 ||
		a.state.Self.Dead || a.inventory.pending || a.shop.pending {
		return
	}
	it, ok := a.state.Inventory[key]
	if !ok || d.active {
		return
	}
	def := a.itemsItem(it.Type, it.TypeID)
	if def == nil || itemtype.Real(it.Type) == 27 {
		return
	}
	if def.ReqOG != 0 {
		a.systemMessage(17, 454, nil) // plays the error cue itself
		return
	}
	d.active, d.source, d.item = true, key, it
	if it.Count > 1 {
		d.stack = true
		a.quantityUI.Open(in.MouseX, in.MouseY, w, h, int(it.Count))
		return
	}
	d.stack = false
	if d.confirm == nil {
		d.confirm, _ = ui.NewMessage(a)
	}
	if d.confirm == nil {
		d.active = false
		return
	}
	d.confirm.Open(a.msg.Get(455), true)
}

func (a *App) emblemCovers(in *platform.Input, w, h int) bool {
	if !a.emblemOpen() {
		return false
	}
	x, y := a.emblemUI.Origin(w, h)
	return pointInside(in, x, y, ui.EmblemWidth, ui.EmblemHeight)
}

func (a *App) sendItemDrop(count uint16) {
	d := &a.drop
	d.active = false
	if a.world == nil || a.state.Self.Dead || a.state.Inventory[d.source] != d.item {
		return
	}
	if err := a.world.Send(proto.ItemDropPacket(d.source[0], d.source[1], count)); err != nil {
		a.backToLoginID(12131)
	}
}

// updateDropDialog is first in the pointer chain so the dialog owns the mouse.
func (a *App) updateDropDialog(in *platform.Input, w, h int) bool {
	d := &a.drop
	if !d.active {
		return false
	}
	if a.state.Self.Dead || a.state.Inventory[d.source] != d.item {
		d.active = false
		if d.confirm != nil {
			d.confirm.Visible = false
		}
		return false
	}
	if d.stack {
		switch n := a.quantityUI.Update(in); {
		case n > 0:
			a.sendItemDrop(uint16(n))
		case n < 0:
			d.active = false
		}
		return true
	}
	switch d.confirm.Update(in, w, h) {
	case ui.MessageOK:
		a.sendItemDrop(1)
	case ui.MessageCancel:
		d.active = false
	}
	return true
}

func (a *App) drawDropDialog(w, h int) {
	d := &a.drop
	if !d.active {
		return
	}
	if d.stack {
		a.quantityUI.Draw(a.sprites, a.font)
	} else if d.confirm != nil {
		d.confirm.Draw(a.sprites, a.font, w, h)
	}
}
