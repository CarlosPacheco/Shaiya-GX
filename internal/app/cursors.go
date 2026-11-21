package app

import (
	"fmt"
	"log"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

// LoadCursors reads lossless PNG artwork. All seven supplied cursors use
// the top-left pixel as their click hotspot, as in the original CUR files.
func (a *App) LoadCursors(window *platform.Window) *platform.Cursors {
	c := platform.NewCursors(window)
	for i := 0; i < 7; i++ {
		path := fmt.Sprintf("Cursor/%02d.png", i+1)
		b, err := a.arc.ReadFile(path)
		if err == nil {
			var cursor img.Texture
			cursor, err = img.DecodePNG(b)
			if err == nil {
				c.Load(i, cursor.RGBA.Pix, cursor.RGBA.W, cursor.RGBA.H, 0, 0)
			}
		}
		if err != nil {
			log.Printf("cursor %s: %v", path, err)
		}
	}
	c.Set(0)
	return c
}

// Native 0x41fde0 picks the cursor every frame from what lies under the
// pointer, in this order: a ground item (pickup), a hostile character or
// monster (attack), an NPC (talk), otherwise the arrow. Its switch has no
// path to the crosshair (07.png): no skill, approach or assignment state ever
// selects it, so GX never shows it. Windows and the bottom UI keep the arrow.
func (a *App) CursorIndex(w, h int) int {
	a.hoveredMob = 0
	a.hoverPicking = true
	defer func() { a.hoverPicking = false }()
	if a.screen != screenWorld || a.wv == nil || a.state == nil {
		return 0
	}
	if a.skills.drag.active || a.inventory.dragging || a.wv.localActionDrag.active {
		return 4
	}
	in := &platform.Input{MouseX: a.lastMouseX, MouseY: a.lastMouseY, Clicked: true}
	if c := a.playerContext; c.open && pointInside(in, c.x, c.y, c.width, c.height) {
		return 0
	}
	if a.barCovered(in, w, h) || a.menuOpen || a.chatOpen || a.wv.mapView.full {
		return 0
	}
	if a.actionBar != nil && a.actionBar.Contains(in.MouseX, in.MouseY, w, h) {
		return 0 // the original never swaps the arrow over the bars
	}
	x, y := 8, 8
	tx, ty := 256, 0
	if a.hud != nil {
		x, y = a.hud.Origin(w, h)
		tx, ty = a.hud.TargetOrigin(w, h)
	}
	if in.MouseY >= h-64 || pointInside(in, x, y, 213, 114) || a.state.Target != 0 && pointInside(in, tx, ty, 256, 64) {
		return 0
	}
	if a.minimap != nil && !a.wv.mapView.compactHidden {
		x, y := a.minimap.Origin(w, h)
		if pointInside(in, x, y, 196, 197) {
			return 0
		}
	}
	if a.pickGroundModel(in, w, h) != 0 {
		return 2
	}
	if id := a.pickMob(in, w, h); id != 0 {
		entity := a.state.Entities[id]
		if entity.Kind == game.KindMob {
			a.hoveredMob, a.hoveredMobX, a.hoveredMobY = id, in.MouseX, in.MouseY
		}
		if entity.Kind == game.KindUser && entity.Shape != nil {
			if own, ok := a.selfSlot(); ok && own.Race/2 == entity.Shape.Race/2 {
				return 0
			}
		}
		return 1
	}
	if a.pickNPC(in, w, h) != 0 {
		return 5
	}
	return 0
}

// Hover uses completed asynchronous depth samples; clicks validate the current
// captured scene exactly, so cursor feedback cannot permit a click through walls.
func (a *App) pickNameVisibleAt(x, y, w, h int, depth float32) bool {
	if a.hoverPicking {
		return a.sprites.NameHoverVisibleAt(x, y, w, h, depth)
	}
	return a.sprites.NameVisibleAt(x, y, w, h, depth)
}
