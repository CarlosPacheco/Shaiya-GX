//go:build integration

package app

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
)

func TestTeleportStoneAgainstServer(t *testing.T) {
	st, world, send, wait := testworld.Enter(t)
	key := [2]uint8{1, 23}
	const stoneID = 65
	item := st.Inventory[key]
	if item.Type != 100 || item.TypeID != stoneID || item.Count < 2 {
		t.Skip("requires two Teleportation Stones in disposable bag slot 23")
	}
	a := inputApp()
	a.state, a.world = st, world
	a.items = map[sdata.ItemKey]*sdata.Item{}
	receivedTeleport := false

	await := func(action uint8, wantOK bool) {
		t.Helper()
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		for {
			select {
			case p, ok := <-world.Packets:
				if !ok {
					t.Fatal("world disconnected")
				}
				ev, err := st.Apply(p)
				if err != nil {
					t.Fatal(err)
				}
				if ev.MapLoadAck {
					send(proto.MapLoadedPacket())
				}
				if ev.Op == proto.CharacterMapTeleport {
					receivedTeleport = true
				}
				if ev.Op != proto.GXTeleportStone {
					continue
				}
				r, err := proto.DecodeTeleportStone(p)
				if err != nil || r.Action != action || r.OK != wantOK {
					t.Fatalf("recorded point result: %+v, %v", r, err)
				}
				a.confirmedTeleportStone(p)
				return
			case <-timer.C:
				t.Fatal("no stone response")
			}
		}
	}
	x, y := teleportOrigin(1024, 768)
	click := func(dx, dy int) {
		a.updateTeleportStone(&platform.Input{Clicked: true, MouseX: x + dx, MouseY: y + dy}, 1024, 768)
	}
	send(proto.TeleportStonePacket(proto.TeleportTravel, 0))
	await(proto.TeleportTravel, false)
	a.useInventoryItem(key)
	await(proto.TeleportOpen, true)
	for i := range proto.TeleportPointCount {
		click(40, 50+i*32)
		click(40, 224)
		await(proto.TeleportSave, true)
		if !a.teleportStone.points[i].Saved {
			t.Fatalf("point %d did not save", i+1)
		}
	}
	if st.Inventory[key].Count != item.Count {
		t.Fatal("saving charged a stone")
	}

	id := st.SelfID
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
	send(proto.SelectCharacterPacket(id))
	wait(func() bool { return st.Phase == game.PhaseInWorld && st.InventoryReady })
	a.useInventoryItem(key)
	await(proto.TeleportOpen, true)
	for i, point := range a.teleportStone.points {
		if !point.Saved {
			t.Fatalf("point %d lost on reselect", i+1)
		}
	}
	savedMap := a.teleportStone.points[3].Map
	if st.Self.IsGM && savedMap != 18 {
		send(proto.ChatPacket(proto.ChannelNormal, false, "/cmove 18"))
		wait(func() bool { return st.MapID == 18 })
	}
	receivedTeleport = false
	click(40, 50+3*32)
	click(220, 224)
	await(proto.TeleportTravel, true)
	if !receivedTeleport || st.MapID != savedMap || a.teleportStone.open || st.Inventory[key].Count != item.Count-1 {
		t.Fatal("successful travel did not close the UI and consume exactly one stone")
	}
	send(proto.TeleportStonePacket(proto.TeleportTravel, 3))
	await(proto.TeleportTravel, false)
	if st.Inventory[key].Count != item.Count-1 {
		t.Fatal("replay consumed a second stone")
	}
}
