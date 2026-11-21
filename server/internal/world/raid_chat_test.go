package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

func TestRaidNoticeReachesTheWholeRaidFromLeaderAndSubLeaderOnly(t *testing.T) {
	leader, leaderWire := potentialSession(t)
	member := func(offset uint32, sameMap bool) (*Session, <-chan []byte) {
		ch := *leader.char
		ch.ID += 700000 + offset
		if !sameMap {
			ch.MapID++
		}
		s, wire := gmWireSession(t, ch, false)
		s.faction, s.items = leader.faction, leader.items
		s.player = RegisterPlayer(ch, nil, false, s.faction, s.out, s.logger)
		t.Cleanup(s.player.Unregister)
		return s, wire
	}
	sub, subWire := member(1, true)
	plain, plainWire := member(2, false) // another map: raid notices ignore the map
	pt := &party{kind: raidKindRaid, leader: 0, subLeader: 1, slots: []uint16{0, 1, 2},
		members: []*ConnectedPlayer{leader.player, sub.player, plain.player}}
	partyMu.Lock()
	for _, p := range pt.members {
		p.party = pt
	}
	partyMu.Unlock()
	t.Cleanup(func() {
		partyMu.Lock()
		defer partyMu.Unlock()
		for _, p := range pt.members {
			p.party = nil
		}
	})
	notice := func(from *Session, text string) []byte {
		frame := proto.ChatPacket(proto.ChannelRaid, false, text)
		from.dispatch(packet.ChatRaid, frame)
		return nil
	}
	expect := func(wire <-chan []byte, want string) {
		t.Helper()
		timeout := time.After(time.Second)
		for {
			select {
			case frame := <-wire:
				if proto.Op(frame) == proto.ChatRaid {
					c, err := proto.DecodeChatByID(frame)
					if err != nil || c.Text != want {
						t.Fatalf("notice %q %v, want %q", c.Text, err, want)
					}
					return
				}
			case <-timeout:
				t.Fatalf("missing raid notice %q", want)
			}
		}
	}
	notice(leader, "Gather at the gate")
	expect(leaderWire, "Gather at the gate")
	expect(subWire, "Gather at the gate")
	expect(plainWire, "Gather at the gate")
	notice(sub, "Follow the leader")
	expect(plainWire, "Follow the leader")
	notice(plain, "I am not allowed")
	select {
	case frame := <-leaderWire:
		if proto.Op(frame) == proto.ChatRaid {
			if c, _ := proto.DecodeChatByID(frame); c.Text == "I am not allowed" {
				t.Fatal("a plain member sent a raid notice")
			}
		}
	case <-time.After(150 * time.Millisecond):
	}
}
