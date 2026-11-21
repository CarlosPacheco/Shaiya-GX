package world

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

func TestChatItemSnapshotPreservesInstanceProperties(t *testing.T) {
	weapon := db.Item{Bag: 0, Slot: 5, Type: 12, TypeID: 108, Quality: 88, Count: 1,
		Gems: [6]uint8{164, 243}, CraftName: "01020304050607080920"}
	apple := db.Item{Bag: 1, Slot: 5, Type: 25, TypeID: 8, Count: 10}
	s := Session{items: []db.Item{weapon, apple}}
	request := proto.WithChatItemSlots(proto.ChatPacket(proto.ChannelNormal, false, "Gear"), [][2]uint8{{0, 5}, {1, 5}})
	snapshots, ok := s.chatItemSnapshots(request, 2)
	if !ok {
		t.Fatal("owned inventory slots rejected")
	}
	s.items[0].Gems = [6]uint8{}
	st := game.New()
	frames := [][]byte{
		chatWithItems(buildChatByID(packet.ChatNormal, 42, "Gear"), snapshots),
		chatWithItems(buildChatByName(packet.ChatGuild, "Priest", "Gear"), snapshots),
		chatWithItems(buildWhisper(0, "Priest", "Gear"), snapshots),
	}
	for _, frame := range frames {
		if int(binary.LittleEndian.Uint16(frame)) != len(frame) {
			t.Fatal("chat frame length mismatch")
		}
		if _, err := st.Apply(frame[2:]); err != nil {
			t.Fatal(err)
		}
		if len(st.Chat) == 0 {
			t.Fatal("chat packet was not delivered")
		}
		line := st.Chat[len(st.Chat)-1]
		item := line.Items[0]
		if line.Text != "Gear" || item.Type != weapon.Type || item.TypeID != weapon.TypeID ||
			item.Gems != weapon.Gems || item.Craft != weapon.CraftName || item.Quality != weapon.Quality ||
			line.Items[1].TypeID != apple.TypeID || line.Items[1].Count != apple.Count {
			t.Fatalf("chat lost its item snapshot: %+v", line)
		}
	}
	for _, slot := range [][2]uint8{{1, 6}, {9, 0}, {0, 24}} {
		p := proto.WithChatItemSlots(proto.ChatPacket(proto.ChannelNormal, false, "Gear"), [][2]uint8{slot})
		if _, ok := s.chatItemSnapshots(p, 2); ok {
			t.Fatalf("invalid attachment accepted: %v", slot)
		}
	}
}

func TestPublicChatFactionAndGMRecipients(t *testing.T) {
	const mapID = 60319
	player := func(id uint32, faction uint8, gm bool) (*Session, <-chan []byte) {
		s, frames := gmWireSession(t, db.Character{ID: id, Name: "ChatProbe", MapID: mapID, PosX: 10, PosZ: 10}, gm)
		s.faction = faction
		s.player = RegisterPlayer(*s.char, nil, gm, faction, s.out, s.logger, s)
		t.Cleanup(s.player.Unregister)
		return s, frames
	}
	light, own := player(990701, FactionLight, false)
	_, ally := player(990702, FactionLight, false)
	_, enemy := player(990703, FactionDark, false)
	_, gm := player(990704, FactionDark, true)
	for _, op := range []proto.Opcode{proto.ChatNormal, proto.ChatMap, proto.ChatShout, proto.ChatWorld} {
		light.handlePublicChat(packet.Opcode(op), "Test", proto.ChatItems{})
		for _, frames := range []<-chan []byte{own, ally, gm} {
			waitWorldPacket(t, frames, op)
		}
	}
	select {
	case frame := <-enemy:
		t.Fatalf("opposing player received public chat %x", frame)
	case <-time.After(80 * time.Millisecond):
	}
}
