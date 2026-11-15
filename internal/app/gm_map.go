package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"log"
)

// mapClick separates a click from panning, including an out-and-back drag.
type mapClick struct {
	active, moved bool
	x, y          int
}

func (c *mapClick) update(in *platform.Input, over bool) bool {
	if in.Pressed {
		*c = mapClick{active: over, x: in.MouseX, y: in.MouseY}
	}
	if c.active {
		dx, dy := in.MouseX-c.x, in.MouseY-c.y
		if dx*dx+dy*dy > 16 || in.Wheel != 0 || in.RightPressed {
			c.moved = true
		}
	}
	clicked := c.active && !c.moved && over && in.Clicked
	if !in.MouseDown {
		c.active = false
	}
	return clicked
}
func (a *App) gmMapTeleport(mapID uint16, view ui.UV, fx, fy float32) {
	if a.world == nil || a.state == nil || !a.state.Self.IsGM {
		return
	}
	// A size of one returns normalized X/Z; map-image V points downward.
	u, z, ok := ui.MapCoordinates(view, fx, fy, 1)
	if !ok {
		return
	}
	if err := a.world.Send(proto.GMMapTeleportPacket(mapID, u, 1-z)); err != nil {
		log.Printf("GM map teleport: %v", err)
	}
}
