package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"testing"
)

func TestInventoryAndStatusStayOpenInEitherOrder(t *testing.T) {
	for _, keys := range []string{"it", "ti"} {
		t.Run(keys, func(t *testing.T) {
			a := inputApp()
			for _, key := range keys {
				a.worldControls(&platform.Input{Chars: []rune{key}}, 1024, 768)
			}
			if !a.inventory.open || !a.status.open {
				t.Fatal("opening a panel closed its neighbor")
			}
			ix, iy := a.inventoryUI.Origin(1024, 768)
			sx, sy := a.statusUI.Origin(1024, 768)
			if ix < sx+ui.StatusWidth && ix+ui.InventoryWidth > sx && iy < sy+ui.StatusHeight && iy+ui.InventoryHeight > sy {
				t.Fatal("default frames hide one another")
			}
			a.worldControls(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
			if a.status.open || !a.inventory.open || a.menuOpen {
				t.Fatal("Escape did not close only the top panel")
			}
		})
	}
}

func TestOverlappingStatusOwnsClickAndTitleDrag(t *testing.T) {
	a := inputApp()
	a.inventory.open, a.status.open = true, true
	a.inventoryUI.Position.Move(400, 100, 1024, 768, ui.InventoryWidth, ui.InventoryHeight)
	a.statusUI.Position.Move(400, 100, 1024, 768, ui.StatusWidth, ui.StatusHeight)
	// The stats expand tab covers an inventory equipment cell.
	a.worldControls(&platform.Input{MouseX: 405, MouseY: 260, Pressed: true, MouseDown: true}, 1024, 768)
	c := a.worldControls(&platform.Input{MouseX: 405, MouseY: 260, Clicked: true}, 1024, 768)
	if !a.statusUI.Expanded || c.Clicked || a.inventory.dragging {
		t.Fatal("covered inventory captured the stats control")
	}
	a.statusUI.Expanded = false
	a.worldControls(&platform.Input{MouseX: 450, MouseY: 115, Pressed: true, MouseDown: true}, 1024, 768)
	if a.windowDrag.position != &a.statusUI.Position {
		t.Fatal("title drag reached covered inventory")
	}
}

func TestPersonalPanelOpeningPreservesMovedLayout(t *testing.T) {
	a := inputApp()
	a.statusUI.Position.Move(100, 60, 1024, 768, ui.StatusWidth, ui.StatusHeight)
	saved := a.statusUI.Position
	for _, key := range "iktb" {
		a.worldControls(&platform.Input{Chars: []rune{key}}, 1024, 768)
	}
	if a.personalPanelMask() != 15 || a.statusUI.Position != saved {
		t.Fatal("panel opening lost another window or saved layout")
	}
}
