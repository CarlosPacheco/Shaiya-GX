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
	if x < 0 || z < 0 || x > size || z > size {
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
	at, ok := nearbyMapDestination(nav, g, mapID, size, x, hint, z)
	if !ok {
		return fmt.Errorf("no safe destination within 32 units")
	}

	s.cancelMove()
	s.cast = nil
	s.syncPlayerSpeed(false)
	s.autoAttack = false
	s.useNpc(0)
	s.applyTeleport(mapID, at[0], at[1], at[2], "GM map teleport", true)
	if s.char.MapID != mapID || math.Abs(float64(s.char.PosX-at[0])) > .001 || math.Abs(float64(s.char.PosZ-at[2])) > .001 {
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
	// this candidate. The caller can try a nearby safe point instead.
	if !ok || !g.PlayerBodyClear(at) || math.Abs(float64(at[0]-x)) > .001 || math.Abs(float64(at[2]-z)) > .001 {
		return mapgeometry.Position{}, false
	}
	return at, true
}

// Prefer the exact click, then the nearest navmesh projection. If its player
// body is obstructed, search bounded expanding rings; never accept a floorless
// point, a blocked mask cell, or coordinates outside the destination map.
func nearbyMapDestination(nav *mapnav.Map, g *mapgeometry.Geometry, mapID uint16, size, x, hint, z float32) (mapgeometry.Position, bool) {
	pick := func(px, pz, radius float32) (mapgeometry.Position, bool) {
		p := mapnav.Point{px, hint, pz}
		if nav != nil {
			var ok bool
			p, ok = nav.Project(p, radius, 1e6)
			if !ok {
				return mapgeometry.Position{}, false
			}
		}
		if (p[0]-x)*(p[0]-x)+(p[2]-z)*(p[2]-z) > 32*32 {
			return mapgeometry.Position{}, false
		}
		if p[0] < 0 || p[2] < 0 || p[0] >= size || p[2] >= size || !IsMapWalkable(mapID, p[0], p[2]) {
			return mapgeometry.Position{}, false
		}
		return exactMapDestination(g, p[0], p[1], p[2])
	}
	if p, ok := pick(x, z, .25); ok {
		return p, true
	}
	if nav != nil {
		if p, ok := pick(x, z, 32); ok {
			return p, true
		}
	}
	for radius := float64(1); radius <= 32; radius++ {
		points := int(math.Ceil(2 * math.Pi * radius))
		for i := 0; i < points; i++ {
			angle := 2 * math.Pi * float64(i) / float64(points)
			if p, ok := pick(x+float32(radius*math.Cos(angle)), z+float32(radius*math.Sin(angle)), .75); ok {
				return p, true
			}
		}
	}
	return mapgeometry.Position{}, false
}
