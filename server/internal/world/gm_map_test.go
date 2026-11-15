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
