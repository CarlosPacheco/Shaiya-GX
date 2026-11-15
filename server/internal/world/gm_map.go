package world

import (
	"fmt"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/mapnav"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func (s *Session) handleGMMapTeleport(p []byte) {
	if s.char == nil {
		return
	}
	if !s.isAdmin {
		s.gmReply("GM: access denied.")
		return
	}
	if tradeFor(s.char.ID) != nil {
		s.gmReply("GM: finish trading before teleporting.")
		return
	}
	mapID, u, v, err := proto.DecodeGMMapTeleport(p)
	if err == nil {
		err = s.gmMapTeleport(mapID, u, v)
	}
	if err != nil {
		s.gmReply("GM: " + err.Error() + ".")
	}
}

func (s *Session) gmMapTeleport(mapID uint16, u, v float32) error {
	size, ok := mapSize(mapID)
	if !ok || size <= 0 || isInstanceZone(mapID) {
		return fmt.Errorf("unknown map")
	}
	x, z := u*size, (1-v)*size
	if x < 0 || z < 0 || x >= size || z >= size || !IsMapWalkable(mapID, x, z) {
		return fmt.Errorf("destination is outside the playable map")
	}
	nav, err := loadMobNavigation(mapID)
	if err != nil || nav == nil {
		return fmt.Errorf("destination collision resources unavailable")
	}
	g, ok := nav.PlayerGeometry()
	if !ok {
		return fmt.Errorf("destination physics unavailable")
	}
	// Use the nearest navigable layer as a height hint, never as an X/Z snap.
	hint := g.Height(x, z)
	if s.char.MapID == mapID {
		hint = s.char.PosY
	}
	point, ok := nav.Project(mapnav.Point{x, hint, z}, .25, 1e6)
	if !ok || math.Abs(float64(point[0]-x)) > .001 || math.Abs(float64(point[2]-z)) > .001 {
		return fmt.Errorf("destination has no walkable floor")
	}
	at, ok := exactMapDestination(g, x, point[1], z)
	if !ok {
		return fmt.Errorf("destination is obstructed")
	}
	s.cancelMove()
	s.cast = nil
	s.syncPlayerSpeed(false)
	s.autoAttack = false
	s.useNpc(0)
	s.applyTeleport(mapID, at[0], at[1], at[2], "GM map teleport", true)
	if s.char.MapID != mapID || math.Abs(float64(s.char.PosX-x)) > .001 || math.Abs(float64(s.char.PosZ-z)) > .001 {
		return fmt.Errorf("destination unavailable")
	}
	s.gmReply(fmt.Sprintf("GM: map %d, X %.2f Y %.2f Z %.2f.", mapID, s.char.PosX, s.char.PosY, s.char.PosZ))
	return nil
}

func exactMapDestination(g *mapgeometry.Geometry, x, hint, z float32) (mapgeometry.Position, bool) {
	y, ok := g.Support(x, hint, z, .65)
	if !ok {
		return mapgeometry.Position{}, false
	}
	at, ok := g.Spawn(mapgeometry.Position{x, y + mapgeometry.FootSkin, z})
	// StandingPosition can push a body away from walls; reject instead of moving
	// the clicked point. The map click always means those exact X/Z coordinates.
	if !ok || math.Abs(float64(at[0]-x)) > .001 || math.Abs(float64(at[2]-z)) > .001 {
		return mapgeometry.Position{}, false
	}
	return at, true
}
