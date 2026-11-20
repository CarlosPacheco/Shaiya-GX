package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestFirecrackerReachesNearbyPlayersOfBothFactions(t *testing.T) {
	s, add := forgeRuleSession(t)
	s.char.Name = "Fireworks"
	mapID := s.char.MapID
	defs := db.MustDefs()
	item := &db.ItemDef{Type: 100, TypeID: 116, Special: specialFirecracker, ReqIg: 2}
	defs.Items[db.ItemKey{Type: 100, TypeID: 116}] = item
	add(db.Item{Type: 100, TypeID: 116, Bag: 1, Slot: 1, Count: 2})
	observer := func(id uint32, faction uint8, mapID uint16, x float32) <-chan []byte {
		receiver, wire := gmWireSession(t, db.Character{ID: id, MapID: mapID, PosX: x}, false)
		player := RegisterPlayer(*receiver.char, nil, false, faction, receiver.out, receiver.logger)
		t.Cleanup(player.Unregister)
		return wire
	}
	light := observer(94101, FactionLight, mapID, 2)
	dark := observer(94102, FactionDark, mapID, 3)
	distant := observer(94103, FactionDark, mapID, firecrackerRadius+1)
	other := observer(94104, FactionLight, mapID+1, 0)
	s.handleUseItem(proto.UseItemPacket(1, 1))
	for _, wire := range []<-chan []byte{light, dark} {
		event, err := proto.DecodeFirecracker(waitWorldPacket(t, wire, proto.GXFirecracker))
		if err != nil || event.Variant != 2 || event.Name != s.char.Name || event.X != s.char.PosX {
			t.Fatalf("incorrect firecracker event: %+v (%v)", event, err)
		}
	}
	s.handleUseItem(proto.UseItemPacket(1, 1))
	if s.items[0].Count != 1 {
		t.Fatal("item consumption or cooldown was incorrect")
	}
	for _, wire := range []<-chan []byte{distant, other, light, dark} {
		select {
		case p := <-wire:
			t.Fatalf("unexpected extra firecracker packet: %x", p)
		case <-time.After(20 * time.Millisecond):
		}
	}
}
