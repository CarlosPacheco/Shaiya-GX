package app

import (
	"log"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// mountItemType is the equipment type of mounts (vehicle slot 13); the item
// Range byte is the vehicle MON record, as the server's 0x0B01 model.
const mountItemType = 42

type mallPreview struct {
	loaded     bool
	appearance scene.Appearance
	mount      int // -1 when the character is shown
	race       uint8
	actor      *scene.Character
	vehicle    *scene.Vehicle
}

// mallPreviewTarget returns the appearance wearing the selected product's
// equipment, or the vehicle record when the product is a mount.
func (a *App) mallPreviewTarget(base scene.Appearance) (scene.Appearance, int) {
	products := a.mall.products[a.mallUI.Category]
	if a.mallUI.Selected < 0 || a.mallUI.Selected >= len(products) {
		return base, -1
	}
	ap := base
	for _, id := range products[a.mallUI.Selected].ItemIDs {
		typ, typeID := uint8(id/1000), uint8(id%1000)
		def := a.itemsItem(typ, typeID)
		if def == nil {
			continue
		}
		if typ == mountItemType {
			return base, int(def.Range)
		}
		slots := inventoryEquipSlots(typ)
		if len(slots) == 0 {
			continue
		}
		model := def.Model
		switch slots[0] {
		case 0:
			ap.Helmet, ap.ShowHelmet = int(model), true
		case 1:
			ap.Upper = model
		case 2:
			ap.Lower = model
		case 3:
			ap.Hand = model
		case 4:
			ap.Foot = model
		case 5:
			ap.WeaponType, ap.WeaponModel, ap.WeaponEnchant = typ, model, 0
		case 6:
			ap.ShieldType, ap.ShieldModel, ap.OffhandEnchant = typ, model, 0
		case 7:
			ap.CloakType, ap.CloakModel = typ, model
		}
	}
	return ap, -1
}

func (a *App) drawMallPreview(w, h int) {
	slot, ok := a.selfSlot()
	if !ok || a.assets == nil || a.mallUI.GiftOpen {
		return
	}
	base := a.appearance(&slot)
	ap, mount := a.mallPreviewTarget(base)
	p := &a.mall.preview
	if !p.loaded || p.appearance != ap || p.mount != mount || p.race != slot.Race {
		*p = mallPreview{loaded: true, appearance: ap, mount: mount, race: slot.Race}
		var err error
		if mount >= 0 {
			if p.vehicle, err = a.assets.BuildVehicle(slot.Race, uint8(mount)); err != nil {
				log.Printf("mall preview mount: %v", err)
			}
		} else if p.actor, err = scene.BuildCharacter(a.assets, ap, "000_normal"); err != nil {
			log.Printf("mall preview: %v", err)
			// A model missing for this race or gender keeps the current look.
			p.actor, _ = scene.BuildCharacter(a.assets, base, "000_normal")
		}
	}
	x, y := a.mallUI.Origin(w, h)
	px, py, pw, ph := ui.MallPreviewRect()
	switch {
	case p.vehicle != nil:
		a.drawMallVehicle(p.vehicle, x+px, y+py, pw, ph, w, h)
	case p.actor != nil:
		// Same framing as the inventory, centred in the wider box.
		fw := ph * 136 / 217
		a.drawEquipmentActor(p.actor, ap, x+px+(pw-fw)/2, y+py+6, fw, ph-12, w, h)
	}
}

// drawMallVehicle frames the whole mount, turned three quarters to the viewer.
func (a *App) drawMallVehicle(v *scene.Vehicle, x, y, width, height, w, h int) {
	lo, hi, ok := v.PreviewBounds()
	if !ok {
		return
	}
	const fov = 34 * math.Pi / 180
	center := lo.Add(hi).Scale(.5)
	radius := hi.Sub(lo).Len() / 2
	distance := radius / float32(math.Sin(fov/2)) * .9
	a.sprites.Flush()
	render.InViewport(x, y, width, height, h, func() {
		s := a.actorPreviewScene()
		s.SetCameraView(render.V3{0, center[1], distance}, render.V3{0, center[1], 0}, render.V3{0, 1, 0})
		s.Proj = render.PerspectiveFovLH(fov, float32(width)/float32(height), .1, 100)
		s.FogStart, s.FogEnd = 10000, 20000
		s.LightDir = render.V3{-1, -1, -1}
		s.CharacterAmbient = 1
		s.Begin()
		yaw := float32(-35 * math.Pi / 180)
		facing := render.V3{float32(math.Sin(float64(yaw))), 0, -float32(math.Cos(float64(yaw)))}
		v.UIActor = true
		v.World = render.Basis(render.V3{-center[0], 0, -center[2]}, facing, render.V3{0, 1, 0})
		v.Update(float32(a.frameDT))
		v.Draw(a.skinner, &s)
		s.End()
	})
	a.sprites.Begin(w, h)
}
