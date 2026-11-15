package scene

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Native 0x428319 selects these libraries; 0x42a145 loads catalogue IDs
// 13, 14, 218, 219. They contain one emitter per weapon family, not sequences.
var itemGlowFiles = [...]string{"Effect/item_L.EFT", "Effect/item_F.EFT", "Effect/item_E.EFT", "Effect/item_W.EFT"}

func (a *Assets) itemGlow(typ uint8, selector int32) *Effect {
	family := effect.WeaponFamily(typ)
	// Native 0x40e5b0 rejects missing emitter indices. Shields have no entry
	// in the original fifteen-family libraries despite their ITM glow flags.
	if family == 0 || selector < 0 || selector >= int32(len(itemGlowFiles)) {
		return nil
	}
	file := itemGlowFiles[selector]
	if !a.itemGlowTried[selector] {
		a.itemGlowTried[selector] = true
		raw, err := a.Arc.ReadFile(file)
		if err == nil {
			a.itemGlowLibraries[selector], err = effect.ParseEFT(raw)
		}
		if err != nil {
			a.Missing[fmt.Sprintf("%s: %v", file, err)] = true
		}
	}
	library := a.itemGlowLibraries[selector]
	index := int(family) - 1
	if library == nil {
		return nil
	}
	if index >= len(library.Effects) {
		a.Missing[fmt.Sprintf("%s: emitter %d unavailable", file, index)] = true
		return nil
	}
	single := *library
	single.Sequences = []effect.Sequence{{Records: []effect.Record{{Effect: int32(index)}}}}
	x, err := loadEffectLibrary(a, fmt.Sprintf("%s#emitter%d", file, index), 0, &single)
	if err != nil {
		a.Missing[err.Error()] = true
		return nil
	}
	if x.Skipped != 0 {
		a.Missing[fmt.Sprintf("%s: emitter %d unsupported", file, index)] = true
		return nil
	}
	// 0x42947b uses world-space births (0x55de50) only for fire, ID14.
	// Other innate glows use local simulation (0x55d270), then the item's
	// current matrix at draw time. Transforming births makes them lag behind.
	x.localParticles = selector != 1
	x.particleWorld = render.Identity()
	return x
}

func (x *Effect) setItemGlowTransform(m render.M4) {
	if x.localParticles {
		x.particleWorld = m
		x.Origin, x.basis = render.V3{}, render.Identity()
	} else {
		x.SetBoneTransform(m)
	}
}
