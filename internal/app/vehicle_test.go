package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestVehicleCanonicalCancellationClearsPendingRequest(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 42
	a.vehicle.pending = true
	a.confirmedVehicle(proto.VehicleStatePacket(proto.VehicleState{ID: 42, State: proto.VehicleOff}))
	if a.vehicle.pending {
		t.Fatal("asynchronous cancellation left mount request pending")
	}
}

func TestVehicleCooldownUsesSecondsIndependentlyOfPotions(t *testing.T) {
	a := inputApp()
	a.clock = 10
	a.inventory.cooldowns = map[uint8]float64{10: 610}
	it := proto.InventoryItem{Type: 42, TypeID: 5}
	if a.itemCooldown(it) != 0 {
		t.Fatal("potion cooldown blocked mount")
	}
	a.vehicle.cooldownEnd = 20
	if a.itemCooldown(it) != 10 || itemCooldownDuration(&sdata.Item{Type: 42, ReqIg: 10}) != 10 {
		t.Fatal("mount ReqIg was treated as potion group")
	}
}

func TestMountAppearanceSoundUsesNativeRaceAndBoardRules(t *testing.T) {
	want := []string{"run_horse-roar01.wav", "run_pumav-roar01.wav", "run_pumav-roar01.wav", "run_baslk-roar01.wav"}
	for race, name := range want {
		if got := mountAppearanceSound(uint8(race), 1); got != name {
			t.Fatal(race, got)
		}
		if mountAppearanceSound(uint8(race), 6) != "" {
			t.Fatal("board played animal roar")
		}
	}
}
