package app

import (
	"log"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/data/emblem"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// The emblem merchant is the NPC with merchant type 8 (Game.exe 0x48e050:
// the NPC's type byte 8 opens window state 0x6d). Its window customizes a cape
// with up to three patterns and colours for a fixed fee.
const emblemMerchant = 8

// emblemPreviewYaw turns the preview toward the back of the character, at
// exactly (GX choice: the original turns it three quarters).
var emblemPreviewYaw = 225 * math.Pi / 180

type emblemState struct {
	npc          uint32
	hasItem      bool
	key          [2]uint8
	selection    [ui.EmblemShapes]int
	colors       [ui.EmblemShapes]uint8
	active       int
	pending      bool
	serial       uint64
	removeSerial uint64
	box          *ui.Message
	preview      *scene.Character
	previewApp   scene.Appearance
	previewMark  [6]uint8
	previewReady bool
}

func (a *App) emblemNPC(e *game.Entity) bool {
	if e == nil || e.NpcType != 1 || a.emblemUI == nil {
		return false
	}
	return a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}].MerchantType == emblemMerchant
}

func (a *App) emblemOpen() bool {
	return a.shop.npcID != 0 && a.emblemNPC(a.state.Entities[a.shop.npcID]) && !a.shop.overlay() && !a.state.Self.Dead
}

// emblemClear is the reset of 0x480a00: no cape, no selection, white colours
// and the front shape current.
func (a *App) emblemClear() {
	v := &a.emblem
	box := v.box
	*v = emblemState{npc: v.npc, box: box}
	for i := range v.selection {
		v.selection[i], v.colors[i] = -1, emblem.DefaultColor
	}
	v.active = 1
}

func (a *App) emblemCounts() (counts [ui.EmblemShapes]int) {
	for i := range counts {
		counts[i] = a.assets.EmblemCount(i)
	}
	return counts
}

// emblemPlace loads a cape into the window (0x47f400 on a drag onto the slot):
// the stored bytes give the shapes (index+1) and colours.
func (a *App) emblemPlace(key [2]uint8, it proto.InventoryItem) {
	v := &a.emblem
	v.hasItem, v.key = true, key
	p := emblem.FromBytes(it.Gems)
	for i, b := range [ui.EmblemShapes]uint8{p.Front, p.Back, p.Emblem} {
		v.selection[i] = int(b) - 1
	}
	v.colors = [ui.EmblemShapes]uint8{p.FrontColor, p.BackColor, p.EmblemColor}
	for i := range v.colors {
		if v.selection[i] < 0 {
			v.colors[i] = emblem.DefaultColor
		}
	}
	v.active = 1
	v.pending = false
}

func (a *App) emblemPattern() emblem.Pattern {
	v := &a.emblem
	return emblem.Pattern{
		Front: uint8(v.selection[0] + 1), Back: uint8(v.selection[1] + 1), Emblem: uint8(v.selection[2] + 1),
		FrontColor: v.colors[0], BackColor: v.colors[1], EmblemColor: v.colors[2],
	}
}

// emblemSync forgets the cape once the carried item changes under the window.
func (a *App) emblemSync() {
	v := &a.emblem
	if !v.hasItem {
		return
	}
	it, ok := a.state.Inventory[v.key]
	if !ok || (it.Type != 24 && it.Type != 39) {
		a.emblemClear()
	}
}

func (a *App) emblemMessage(id int) { a.emblemText(a.msg.Get(id)) }

func (a *App) emblemText(text string) {
	v := &a.emblem
	if v.box == nil {
		v.box, _ = ui.NewMessage(a)
	}
	if v.box != nil {
		v.box.Open(text, false)
	}
}

func (a *App) emblemView() *ui.EmblemView {
	v := &a.emblem
	view := &ui.EmblemView{Selection: v.selection, Colors: v.colors, Counts: a.emblemCounts(), Active: v.active,
		HasItem: v.hasItem, Busy: v.pending || a.shop.pending || a.inventory.pending, Cost: emblem.Fee,
		Gold: ui.GoldText(emblem.Fee) + " " + a.interfaceText(0x146a)}
	for i := range view.Textures {
		if v.selection[i] >= 0 {
			view.Textures[i] = a.assets.EmblemTexture(i, uint8(v.selection[i]+1))
		}
	}
	return view
}

func (a *App) updateEmblem(in *platform.Input, w, h int, e *game.Entity) bool {
	v := &a.emblem
	if v.npc != e.ID {
		v.npc = e.ID
		a.emblemClear()
		if !a.inventory.open {
			a.toggleInventory()
		}
	}
	if v.box == nil {
		v.box, _ = ui.NewMessage(a)
	}
	a.emblemSync()
	x, y := a.emblemUI.Origin(w, h)
	captured := panelPointer(in, x, y, ui.EmblemWidth, ui.EmblemHeight)
	if v.box != nil && v.box.Visible {
		v.box.Update(in, w, h)
		return true
	}
	if v.pending {
		if a.state.EmblemSerial != v.serial || a.state.EmblemRemoveSerial != v.removeSerial {
			v.pending = false
			a.emblemResult()
		}
	}
	view := a.emblemView()
	act := a.emblemUI.Update(in, w, h, view)
	if act.Close || in.CloseOnEscape() {
		if err := a.world.Send(proto.NpcTalkPacket(0)); err != nil {
			a.backToLoginID(12011)
			return true
		}
		a.shop.npcID = 0
		v.npc = 0
		a.emblemClear()
		return true
	}
	switch {
	case act.Arrow >= 0 && v.hasItem:
		shape, next := act.Arrow/2, act.Arrow%2 == 1
		count := view.Counts[shape]
		if next {
			v.selection[shape] = min(v.selection[shape]+1, count-1)
		} else {
			v.selection[shape] = max(v.selection[shape]-1, -1)
		}
		v.active = shape + 1
	case act.Frame != 0:
		v.active = act.Frame
	case act.Color >= 0 && v.active > 0:
		v.colors[v.active-1] = uint8(act.Color)
	case act.Apply:
		a.emblemApply()
	case act.Remove:
		a.emblemRemove()
	}
	return captured
}

// emblemApply is the first button of 0x47f400: an empty window asks for a
// cape (SysMsg 5359), too little gold gives 5361 and otherwise 0x0803 goes out.
func (a *App) emblemApply() {
	v := &a.emblem
	if !v.hasItem {
		a.emblemMessage(5359)
		return
	}
	if a.state.Details.Money < emblem.Fee {
		a.emblemMessage(5361)
		return
	}
	if a.world == nil {
		return
	}
	v.pending, v.serial, v.removeSerial = true, a.state.EmblemSerial, a.state.EmblemRemoveSerial
	if err := a.world.Send(proto.EmblemApplyPacket(v.key[0], v.key[1], a.emblemPattern().Bytes())); err != nil {
		a.backToLoginID(12056)
	}
}

// emblemRemove is the second button: 0x0804 for the cape in the window.
func (a *App) emblemRemove() {
	v := &a.emblem
	if !v.hasItem || a.world == nil {
		return
	}
	v.pending, v.serial, v.removeSerial = true, a.state.EmblemSerial, a.state.EmblemRemoveSerial
	if err := a.world.Send(proto.EmblemRemovePacket(v.key[0], v.key[1])); err != nil {
		a.backToLoginID(12056)
	}
}

// emblemResult shows the server's answer. A refusal for gold uses the same
// message as the local check; a success shows the stored pattern.
func (a *App) emblemResult() {
	v := &a.emblem
	if a.state.EmblemRemoveSerial != v.removeSerial {
		a.emblemPlace(v.key, a.state.Inventory[v.key])
		a.emblemText(a.tr(12343))
		return
	}
	if a.state.LastEmblem.Result == 0 {
		a.emblemMessage(5361)
		return
	}
	a.emblemPlace(v.key, a.state.Inventory[v.key])
	a.emblemText(a.tr(12344))
}

// updateEmblemDrag drops a dragged cape on the item slot (0x47f400 requires
// the dragged item to be type 24 or 39 and the character level 15 or more).
func (a *App) updateEmblemDrag(in *platform.Input, w, h int) bool {
	if !a.emblemOpen() || !a.inventory.dragging || !a.inventory.selected || !in.Clicked || a.shop.pending || a.inventory.pending {
		return false
	}
	if a.inventory.open && a.inventoryUI != nil {
		ix, iy := a.inventoryUI.Origin(w, h)
		if pointInside(in, ix, iy, ui.InventoryWidth, ui.InventoryHeight) {
			return false
		}
	}
	x, y := a.emblemUI.Origin(w, h)
	sx, sy, sw, sh := ui.EmblemItemSlot()
	if !pointInside(in, x+sx, y+sy, sw, sh) {
		return false
	}
	key := a.inventory.source
	it, ok := a.state.Inventory[key]
	if ok && (it.Type == 24 || it.Type == 39) {
		if slot, found := a.selfSlot(); found && slot.Level < emblem.MinLevel {
			a.emblemMessage(5358)
		} else {
			a.emblemPlace(key, it)
		}
		a.inventory.dragging, a.inventory.selected = false, false
		return true
	}
	return false
}

func (a *App) drawEmblem(w, h int) {
	v := &a.emblem
	a.emblemUI.Draw(a.sprites, a.font, w, h, a.emblemView())
	x, y := a.emblemUI.Origin(w, h)
	a.clearTooltipUnderPanel(x, y, ui.EmblemWidth, ui.EmblemHeight)
	if v.hasItem {
		if it, ok := a.state.Inventory[v.key]; ok {
			if d := a.itemsItem(it.Type, it.TypeID); d != nil {
				a.drawItemIcon(d, x+0x230, y+0x51, 32, white, &it)
			}
		}
		a.drawEmblemPreview(w, h, x, y)
	}
	if v.box != nil && v.box.Visible {
		v.box.Draw(a.sprites, a.font, w, h)
	}
}

// drawEmblemPreview shows the character wearing the cape with the pattern being
// edited, turned three quarters toward the back like 0x47f120.
func (a *App) drawEmblemPreview(w, h, wx, wy int) {
	v := &a.emblem
	slot, ok := a.selfSlot()
	if !ok {
		return
	}
	it := a.state.Inventory[v.key]
	slot.EquipType[7], slot.EquipTypeID[7] = it.Type, it.TypeID
	slot.Cloak = a.emblemPattern().Bytes()
	ap := a.appearance(&slot)
	ap.CloakType = it.Type
	pattern := ap.CloakEmblem
	// Editing a pattern or colour only swaps the decals, so the preview keeps
	// its pose and cloth motion; anything else rebuilds the character.
	ap.CloakEmblem = [6]uint8{}
	if v.previewReady && ap == v.previewApp && v.preview != nil {
		if pattern != v.previewMark {
			v.previewMark = pattern
			v.preview.SetCloakEmblem(pattern)
		}
	} else {
		v.previewReady, v.previewApp, v.previewMark = true, ap, pattern
		ap.CloakEmblem = pattern
		var err error
		v.preview, err = scene.BuildCharacter(a.assets, ap, "000_normal")
		if err != nil {
			log.Printf("emblem preview: %v", err)
		}
	}
	if v.preview == nil {
		return
	}
	px, py, pw, ph := ui.EmblemPreviewArea()
	a.sprites.Flush()
	render.InViewport(wx+px, wy+py, pw, ph, h, func() {
		s := a.actorPreviewScene()
		centerY, distance := v.preview.InventoryFrame(34*math.Pi/180, ap.HeightScale())
		// Closer and higher than the inventory framing: the cape is the subject.
		centerY, distance = centerY*1.22, distance*.62
		s.SetCameraView(render.V3{0, centerY, distance}, render.V3{0, centerY, 0}, render.V3{0, 1, 0})
		s.Proj = render.PerspectiveFovLH(34*math.Pi/180, float32(pw)/float32(ph), .1, 100)
		s.FogStart, s.FogEnd = 10000, 20000
		s.LightDir = render.V3{-1, -1, -1}
		s.CharacterAmbient = 1
		s.Begin()
		scale := ap.ScaleMatrix()
		yaw := emblemPreviewYaw
		forward := render.V3{float32(math.Sin(yaw)), 0, -float32(math.Cos(yaw))}
		v.preview.World = scale.Mul(render.Basis(render.V3{}, forward, render.V3{0, 1, 0}))
		v.preview.Update(float32(a.frameDT))
		v.preview.Draw(a.skinner, &s)
		s.End()
	})
	a.sprites.Begin(w, h)
}
