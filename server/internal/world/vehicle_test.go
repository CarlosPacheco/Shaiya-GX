package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
	"time"
)

func vehicleFixture(t *testing.T, id uint32, x float32) *Session {
	t.Helper()
	s, _ := gmWireSession(t, db.Character{ID: id, MapID: 60081, Race: 0, Class: 0, Level: 60, Mode: 2, CurrentHP: 100, MaxHP: 100, PosX: x}, false)
	s.player = RegisterPlayer(*s.char, nil, false, 0, s.out, nil)
	t.Cleanup(s.player.Unregister)
	return s
}
func TestVehicleCastCooldownAndItemIdentity(t *testing.T) {
	old := db.MustDefs()
	t.Cleanup(func() { db.SetDefsForTest(old) })
	d := &db.ItemDef{Type: 42, TypeID: 1, Range: 0, Speed: 4, AttackTime: 1, ReqIg: 20, Classes: [6]bool{true, true, true, true, true, true}}
	db.SetDefsForTest(&db.Defs{Items: map[db.ItemKey]*db.ItemDef{{Type: 42, TypeID: 1}: d}})
	s := vehicleFixture(t, 990081, 0)
	s.items = []db.Item{{UID: 1, Bag: 0, Slot: 13, Type: 42, TypeID: 1, Count: 1}}
	s.handleUseVehicle()
	if v := s.player.vehicleState(); v.State != proto.VehicleCasting || v.CastMS < 1900 {
		t.Fatal("missing two-second cast", v)
	}
	s.tickVehicle(time.Now().Add(time.Second))
	if s.mounted {
		t.Fatal("mounted too early")
	}
	s.cancelVehicleCast()
	if s.player.vehicleState().State != proto.VehicleOff {
		t.Fatal("cast did not cancel")
	}
	s.handleUseVehicle()
	s.buffs = []ActiveBuff{{Effect: typeEffectDebuff, Detail: detailStop}}
	s.tickVehicle(time.Now().Add(3 * time.Second))
	if s.mounted || s.player.vehicleState().State != proto.VehicleOff {
		t.Fatal("root completed mount preparation")
	}
	s.buffs = nil
	s.handleUseVehicle()
	s.items[0].UID = 2
	s.tickVehicle(time.Now().Add(3 * time.Second))
	if s.mounted {
		t.Fatal("changed item completed cast")
	}
	s.handleUseVehicle()
	end := time.Now().Add(3 * time.Second)
	s.tickVehicle(end)
	if v := s.player.vehicleState(); v.State != proto.VehicleDriver || v.Model != 0 || s.mountSpeed() != 7 {
		t.Fatal("MON zero failed", v)
	}
	s.dismount()
	s.handleUseVehicle()
	if s.player.vehicleState().State != proto.VehicleOff {
		t.Fatal("ReqIg bypassed")
	}
	vehicleMu.Lock()
	vehicleCooldowns[s.char.ID] = time.Now().Add(-time.Second)
	vehicleMu.Unlock()
	d.Speed = 5
	s.handleUseVehicle()
	s.tickVehicle(time.Now().Add(3 * time.Second))
	if s.mountSpeed() != 8 {
		t.Fatal("extremely high speed missing")
	}
}
func TestVehicleInvitationRechecksRangeAndDeparture(t *testing.T) {
	driver := vehicleFixture(t, 990082, 0)
	rider := vehicleFixture(t, 990083, 2)
	vehicleMu.Lock()
	driver.player.ride.state = proto.VehicleState{ID: driver.char.ID, State: proto.VehicleDriver, Driver: driver.char.ID, Race: 0, Model: 1, Seats: 2, Speed: 5}
	vehicleMu.Unlock()
	driver.handleVehicleInvite(proto.VehicleInvitePacket(rider.char.ID))
	rider.char.PosX = 30
	rider.syncPlayer()
	rider.handleVehicleAnswer(proto.VehicleAnswerPacket(true))
	if rider.player.vehicleState().State != proto.VehicleOff || rider.char.PosX != 30 {
		t.Fatal("distant acceptance teleported passenger")
	}
	rider.char.PosX = 2
	rider.syncPlayer()
	driver.handleVehicleInvite(proto.VehicleInvitePacket(rider.char.ID))
	rider.handleVehicleAnswer(proto.VehicleAnswerPacket(false))
	if driver.player.vehicleState().Passenger != 0 {
		t.Fatal("declined request attached passenger")
	}
	driver.handleVehicleInvite(proto.VehicleInvitePacket(rider.char.ID))
	rider.handleVehicleAnswer(proto.VehicleAnswerPacket(true))
	if v := rider.player.vehicleState(); v.State != proto.VehiclePassenger || v.Driver != driver.char.ID || v.Race != 0 {
		t.Fatal("passenger metadata", v)
	}
	if driver.player.vehicleState().Passenger != rider.char.ID {
		t.Fatal("driver missing passenger")
	}
	rider.handleMotion(proto.MotionPacket(proto.MotionJump))
	if rider.player.vehicleState().State != proto.VehicleOff || driver.player.vehicleState().Passenger != 0 {
		t.Fatal("jump did not detach")
	}
	driver.handleVehicleInvite(proto.VehicleInvitePacket(rider.char.ID))
	rider.handleVehicleAnswer(proto.VehicleAnswerPacket(true))
	clearVehicle(driver.player)
	if rider.player.vehicleState().State != proto.VehicleOff {
		t.Fatal("driver departure left passenger attached")
	}
}

func TestMountMetadataDoesNotIncreaseFootStats(t *testing.T) {
	var stats EquipmentStats
	stats.addItemDef(42, &db.ItemDef{Speed: 5, AttackTime: 2})
	if stats.MoveSpeed != 0 || stats.AttackSpeed != 0 {
		t.Fatal("mount metadata leaked into foot stats", stats)
	}
}

func TestVehicleWireStateUsesServerFrame(t *testing.T) {
	s, packets := gmWireSession(t, db.Character{ID: 990084, MapID: 60081, CurrentHP: 100}, false)
	s.player = RegisterPlayer(*s.char, nil, false, 0, s.out, nil)
	t.Cleanup(s.player.Unregister)
	publishVehicle(s.player)
	select {
	case raw := <-packets:
		value, err := proto.DecodeVehicleState(raw)
		if err != nil || value.ID != s.char.ID {
			t.Fatal("bad canonical frame", value, err)
		}
	case <-time.After(time.Second):
		t.Fatal("missing vehicle state frame")
	}
}

func TestVehicleRangeIsTenUnits(t *testing.T) {
	a := vehicleFixture(t, 990084, 0)
	near := vehicleFixture(t, 990085, 9.5)
	far := vehicleFixture(t, 990086, 10.5)
	if !vehicleNear(a.player, near.player) || vehicleNear(a.player, far.player) {
		t.Fatal("go-together and boarding range is 10 units")
	}
}

func TestMountsStayOutOfDungeonsButCrossFieldMapsByPortal(t *testing.T) {
	old := db.MustDefs()
	t.Cleanup(func() { db.SetDefsForTest(old) })
	d := &db.ItemDef{Type: 42, TypeID: 1, Range: 0, Speed: 4, AttackTime: 1, ReqIg: 20, Classes: [6]bool{true, true, true, true, true, true}}
	db.SetDefsForTest(&db.Defs{Items: map[db.ItemKey]*db.ItemDef{{Type: 42, TypeID: 1}: d}})
	const field, dungeon = uint16(60082), uint16(60083)
	oldField, hadField := mapConfig.Zones[field]
	oldDungeon, hadDungeon := mapConfig.Zones[dungeon]
	mapConfig.Zones[field], mapConfig.Zones[dungeon] = ZoneConfig{}, ZoneConfig{Dungeon: true}
	t.Cleanup(func() {
		if hadField {
			mapConfig.Zones[field] = oldField
		} else {
			delete(mapConfig.Zones, field)
		}
		if hadDungeon {
			mapConfig.Zones[dungeon] = oldDungeon
		} else {
			delete(mapConfig.Zones, dungeon)
		}
	})
	s := vehicleFixture(t, 990082, 0)
	s.items = []db.Item{{UID: 1, Bag: 0, Slot: 13, Type: 42, TypeID: 1, Count: 1}}
	s.char.MapID = dungeon
	s.handleUseVehicle()
	if s.player.vehicleState().State != proto.VehicleOff {
		t.Fatal("a mount started casting inside a dungeon")
	}
	s.char.MapID = field
	s.handleUseVehicle()
	s.tickVehicle(time.Now().Add(3 * time.Second))
	if !s.mounted {
		t.Fatal("mount did not start on a field map")
	}
	if !s.keepsMountAcrossTeleport(field, "portal teleport") || !s.keepsMountAcrossTeleport(field, "explicit portal teleport") {
		t.Fatal("a rider must keep the mount through a portal to a field map")
	}
	if s.keepsMountAcrossTeleport(dungeon, "portal teleport") {
		t.Fatal("a rider entering a dungeon must be left on foot")
	}
	if s.keepsMountAcrossTeleport(field, "resurrection") {
		t.Fatal("only portals keep the mount")
	}
}
