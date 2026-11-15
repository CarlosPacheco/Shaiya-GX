package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
)

func TestGMMapPacketRequiresRole(t *testing.T) {
	s, p := gmWireSession(t, db.Character{ID: 990045, MapID: 1, PosX: 42, PosY: 8, PosZ: 36}, false)
	old := *s.char
	s.handleGMMapTeleport(proto.GMMapTeleportPacket(1, .5, .5))
	if r := gmResponse(t, p); r.GM || r.Message != "GM: access denied." {
		t.Fatal(r)
	}
	if *s.char != old {
		t.Fatal("unauthorized map movement")
	}
}
func TestExactMapDestinationPreservesCoordinatesAndRequiresFloor(t *testing.T) {
	s, _ := playerPhysicsFixture(t)
	g := s.movement.geometry
	at, ok := exactMapDestination(g, 8.125, 0, 5.75)
	if !ok || at[0] != 8.125 || at[2] != 5.75 || at[1] > .15 {
		t.Fatal(at, ok)
	}
	if _, ok := exactMapDestination(g, 100, 0, 100); ok {
		t.Fatal("missing floor accepted")
	}
}

func TestMapTeleportSelectsNearbySafeFloorInsteadOfRejectingWall(t *testing.T) {
	s, _ := playerPhysicsFixture(t)
	// A live navmesh excludes the inside of this solid; mirror that in the
	// synthetic map mask while checking the shared player body clearance.
	mask := make([]byte, 32*32/8)
	for z := 0; z < 32; z++ {
		for _, x := range []int{10, 11} {
			at := z*32 + x
			mask[at/8] |= 1 << uint(at%8)
		}
	}
	gridsMu.Lock()
	old, hadGrid := grids[s.char.MapID]
	gridsMu.Unlock()
	RegisterMapGrid(s.char.MapID, 32, mask)
	t.Cleanup(func() {
		gridsMu.Lock()
		defer gridsMu.Unlock()
		if hadGrid {
			grids[s.char.MapID] = old
		} else {
			delete(grids, s.char.MapID)
		}
	})

	g := s.movement.geometry
	p, ok := nearbyMapDestination(nil, g, s.char.MapID, 30, 9.9, 0, 5)
	if !ok || p[0] >= 9.8 && p[0] <= 11.2 || p[1] > .15 {
		t.Fatal("blocked click did not find adjacent floor", p, ok)
	}
	exact, ok := nearbyMapDestination(nil, g, s.char.MapID, 30, 8.125, 0, 5.75)
	if !ok || exact[0] != 8.125 || exact[2] != 5.75 {
		t.Fatal("valid click changed", exact, ok)
	}
	if p, ok := nearbyMapDestination(nil, g, s.char.MapID, 200, 100, 0, 100); ok {
		t.Fatal("teleported beyond bounded search or onto missing floor", p)
	}
}
