package app

import (
	"log"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// Appearance and sex stones open the window of Game.exe state 0x14 (special
// 44, sex locked) or 0x15 (special 45). SysMsg 836 confirms before 0x226.
type appearanceEdit struct {
	source       [2]uint8
	item         proto.InventoryItem
	character    uint32
	confirming   bool
	actor        *scene.Character
	built        scene.Appearance
	synced       bool
	syncedValues [4]uint8
}

func (a *App) openAppearanceEditor(source [2]uint8) bool {
	item, ok := a.state.Inventory[source]
	if !ok || source[0] == 0 {
		return false
	}
	def := a.itemsItem(item.Type, item.TypeID)
	if def == nil || (def.Special != 44 && def.Special != 45) {
		return false
	}
	switch item.Type {
	case 25, 44, 100, 101, 102:
	default:
		return false
	}
	slot, ok := a.selfSlot()
	if !ok || item.Count == 0 {
		return false
	}
	a.appearanceEdit = &appearanceEdit{source: source, item: item, character: slot.ID}
	a.appearanceUI.Open([4]uint8{slot.Hair, slot.Face, slot.Height, slot.Gender}, def.Special == 45)
	a.refreshAppearanceCounts()
	a.syncSelfAppearance()
	return true
}

// refreshAppearanceCounts bounds hair and face by the MLT lists of the body
// the current sex selects, and drops a style the other sex does not have.
func (a *App) refreshAppearanceCounts() {
	slot, ok := a.selfSlot()
	if !ok || a.assets == nil {
		return
	}
	u := a.appearanceUI
	slot.Gender = u.Values[3]
	hair, face := a.assets.StyleCounts(a.appearance(&slot))
	u.Counts = [2]int{hair, face}
	if int(u.Values[0]) >= hair {
		u.Values[0] = 0
	}
	if int(u.Values[1]) >= face {
		u.Values[1] = 0
	}
}

func (a *App) closeAppearanceEditor() {
	a.appearanceEdit = nil
	if a.wv != nil && a.assets != nil {
		a.refreshSelf(a.wv) // restores the real look and helmet
	}
	if a.appearanceConfirm != nil {
		a.appearanceConfirm.Visible = false
	}
}

func (a *App) updateAppearanceEditor(in *platform.Input, w, h int) {
	e := a.appearanceEdit
	// 0x462d80: a vanished item ends the window with SysMsg 239.
	if a.state == nil || a.state.Self.Dead || a.state.SelfID != e.character {
		a.closeAppearanceEditor()
		return
	}
	if a.state.Inventory[e.source] != e.item {
		a.closeAppearanceEditor()
		a.systemMessage(17, 239, nil)
		return
	}
	if e.confirming {
		switch a.appearanceConfirm.Update(in, w, h) {
		case ui.MessageOK:
			d := a.appearanceUI.Values
			a.closeAppearanceEditor()
			if err := a.world.Send(proto.ChangeAppearancePacket(e.source[0], e.source[1], d[0], d[1], d[2], d[3])); err != nil {
				a.backToLoginID(12008)
				return
			}
			a.inventory.pending, a.inventory.deadline = true, a.clock+10
		case ui.MessageCancel:
			e.confirming = false
			a.appearanceConfirm.Visible = false
		}
		return
	}
	a.refreshAppearanceCounts()
	a.syncSelfAppearance()
	switch a.appearanceUI.Update(in, w, h) {
	case ui.MessageCancel:
		a.closeAppearanceEditor()
	case ui.MessageOK:
		e.confirming = true
		a.appearanceConfirm.Open(a.msg.Get(836), true)
	}
}

// drawAppearancePreview follows 0x4631f0: the player's model, helmet off, is
// drawn orthographically at 500 px per unit (430 for race 3) with its feet far
// below the window, so the clip shows head and shoulders.
func (a *App) drawAppearancePreview(w, h int) {
	e := a.appearanceEdit
	slot, ok := a.selfSlot()
	if e == nil || !ok {
		return
	}
	d := a.appearanceUI.Values
	slot.Hair, slot.Face, slot.Height, slot.Gender = d[0], d[1], d[2], d[3]
	ap := a.appearance(&slot)
	// Height shows only on the world character, never in this window.
	ap.Helmet, ap.ShowHelmet, ap.Height = -1, false, 0
	if e.actor == nil || ap != e.built {
		actor, err := scene.BuildCharacter(a.assets, ap, "000_normal")
		if err != nil {
			log.Printf("appearance preview: %v", err)
			return
		}
		e.actor, e.built = actor, ap
	}
	x, y := a.appearanceUI.Origin(w, h)
	scale, tx, ty := appearancePreviewPlacement(ap.Race, ap.Gender)
	left, top := x+ui.AppearanceViewX, y+ui.AppearanceViewTop
	vw, vh := ui.AppearanceViewRight-ui.AppearanceViewX, ui.AppearanceViewBottom-ui.AppearanceViewTop
	a.sprites.Flush()
	render.InViewport(left, top, vw, vh, h, func() {
		s := a.actorPreviewScene()
		s.SetCameraView(render.V3{}, render.V3{0, 0, 1}, render.V3{0, 1, 0})
		s.Proj = render.OrthoOffCenterLH(float32(left), float32(left+vw), float32(h-top-vh), float32(h-top), 1, 400)
		s.FogStart, s.FogEnd = 10000, 20000
		s.LightDir = render.V3{-1, -1, -1}
		s.CharacterAmbient = 1
		s.Begin()
		m := render.Identity()
		m[0], m[5], m[10] = scale, scale, scale
		e.actor.World = m.Mul(render.Basis(render.V3{float32(x) + tx, float32(h-y) - ty, 150}, render.V3{0, 0, 1}, render.V3{0, 1, 0}))
		e.actor.Update(float32(a.frameDT))
		e.actor.Draw(a.skinner, &s)
		s.End()
	})
	a.sprites.Begin(w, h)
}

// appearancePreviewPlacement holds the per-race scale and offsets of 0x4631f0:
// X from the window's left, Y as the distance of the feet below its top.
func appearancePreviewPlacement(race, gender uint8) (scale, tx, ty float32) {
	scale, tx, ty = 500, 0x7d, 0x3a2 // race 1 and the default
	female := gender != 0
	switch race {
	case 0:
		tx = 0x96
		if female {
			ty = 0x384
		}
	case 1:
		if female {
			ty = 0x384
		}
	case 2:
		ty = 0x38e
		if female {
			ty = 0x370
		}
	case 3:
		scale, tx = 430, 0x87
		if female {
			tx, ty = 0x69, 0x370
		}
	}
	return
}

func (a *App) appearanceChanged(id uint32) {
	pos, ok := a.effectPosition(id)
	if !ok {
		return
	}
	a.playSoundWithin("ch_maready02.wav", pos[0], pos[2], 10)
	if x := a.loadSkillEffect("Effect/DA_target_commercialB.EFT", 0); x != nil {
		if m, ok := a.effectTargetMatrix(id); ok {
			x.SetTransform(m)
		}
		a.wv.impacts = append(a.wv.impacts, skillImpact{target: id, instance: x, authoredLifetime: true})
	}
}

// editedSelfAppearance applies the window's draft (helmet off) to the normal
// character, so every change, height included, previews on the world model.
func (a *App) editedSelfAppearance(ap scene.Appearance) scene.Appearance {
	if a.appearanceEdit == nil {
		return ap
	}
	d := a.appearanceUI.Values
	ap.Hair, ap.Face, ap.Height, ap.Gender = d[0], d[1], d[2], d[3]
	ap.Helmet, ap.ShowHelmet = -1, false
	return ap
}

// syncSelfAppearance rebuilds the world model only when the draft changes.
func (a *App) syncSelfAppearance() {
	e := a.appearanceEdit
	if e == nil || a.wv == nil || a.wv.self == nil || a.assets == nil {
		return
	}
	if d := a.appearanceUI.Values; !e.synced || e.syncedValues != d {
		e.synced, e.syncedValues = true, d
		a.refreshSelf(a.wv)
	}
}
