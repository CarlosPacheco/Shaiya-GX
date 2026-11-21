package world

import (
	"path/filepath"
	"testing"
)

func TestCloronPortalVolumesAndArrival(t *testing.T) {
	catalog, err := readPortalVolumes(filepath.Join("..", "..", "data", "game", "PortalBounds.json"))
	if err != nil {
		t.Fatal(err)
	}
	returnPortal := []MapPortal{{Index: 1, TargetMapID: 13, PosX: 210.14938, PosY: .501065, PosZ: 484.14258}}
	catalog.apply(12, returnPortal)
	if returnPortal[0].volume == nil {
		t.Fatal("missing C1 return portal volume")
	}
	if returnPortal[0].Contains(209.94, .546, 480.29) || returnPortal[0].Contains(209.94, .546, 480.09) {
		t.Fatal("arrival or movement away from the gate triggered return to C2")
	}
	if !returnPortal[0].Contains(209.94, .546, 482) {
		t.Fatal("walking into the original portal did not trigger return to C2")
	}

	forwardPortal := []MapPortal{{Index: 1, TargetMapID: 14, PosX: 74.425735, PosY: 1.0941014, PosZ: 71.48961, TargetX: 205, TargetY: 6, TargetZ: 8}}
	catalog.apply(13, forwardPortal)
	p := forwardPortal[0]
	if !p.Contains(74.42, .91055, 71.25) || p.TargetX != 205 || p.TargetY != 15.7 || p.TargetZ != 8 {
		t.Fatalf("C2 portal has incorrect trigger or arrival: %+v", p)
	}
}

func TestPortalVolumeKeepsCustomRoutes(t *testing.T) {
	portal := []MapPortal{{Index: 1, TargetMapID: 77, PosX: 10, PosY: 20, PosZ: 30, TargetY: 8}}
	catalog := portalVolumeCatalog{12: {1: {Target: 14, Min: [3]float32{9, 19, 29}, Max: [3]float32{11, 21, 31}}}}
	catalog.apply(12, portal)
	if portal[0].volume != nil || !portal[0].Contains(14, 20, 30) || portal[0].TargetY != 8 {
		t.Fatalf("metadata changed a custom route: %+v", portal[0])
	}
}
