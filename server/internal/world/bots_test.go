package world

import (
	"net"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestWorldBotsNativeTrafficAndIsolation(t *testing.T) {
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	link, err := openBotWire(listener)
	if err != nil {
		t.Fatal(err)
	}
	defer link.close()
	ch := db.Character{ID: 701, MapID: 7, CurrentHP: 9000}
	s1, p1 := gmWireSession(t, ch, true)
	ch.ID++
	s2, p2 := gmWireSession(t, ch, false)
	v1 := &ConnectedPlayer{CharacterID: 701, out: s1.out}
	v2 := &ConnectedPlayer{CharacterID: 702, out: s2.out}
	a := &worldBotArena{bots: []worldBot{
		{wire: link, char: db.Character{ID: 0x60000001, CurrentHP: botHealth}},
		{faction: 1, char: db.Character{ID: 0x60000002, CurrentHP: 1}},
		{char: db.Character{ID: 0x60000003, CurrentHP: 1200}},
	}, visible: map[*ConnectedPlayer][]bool{v1: {true, true, true}, v2: {true, true, true}}}
	expect := func(op proto.Opcode) {
		t.Helper()
		for _, observer := range []<-chan []byte{p1, p2} {
			select {
			case body := <-observer:
				if got := proto.Op(body); got != op {
					t.Fatalf("got %x, expected %x", got, op)
				}
			case <-time.After(time.Second):
				t.Fatalf("missing %x", op)
			}
		}
	}
	now := time.Now()
	a.applyRequest(0, proto.MovePacket(1, 32, 5, 2, 6), -1, now)
	expect(proto.CharacterMove)
	a.applyRequest(0, proto.UserAttackPacket(v2.CharacterID), 1, now)
	if a.bots[1].char.CurrentHP != 1 {
		t.Fatal("real player target ID matched synthetic opponent")
	}
	a.applyRequest(0, proto.UserAttackPacket(a.bots[1].char.ID), 1, now)
	expect(proto.CharacterUserAutoAttack)
	expect(proto.CharacterDeath)
	a.advance(1, now.Add(4*time.Second))
	expect(proto.CharacterReviveInit)
	expect(proto.CharacterResurrect)
	expect(proto.CharacterReviveHitpoints)
	if a.bots[1].char.CurrentHP != botHealth {
		t.Fatal("bot did not revive")
	}
	a.bots[0].skills = []*db.SkillDef{{SkillID: 14, SkillLevel: 1, TypeEffect: typeEffectHeal}}
	a.applyRequest(0, proto.UseSkillPacket(0, a.bots[2].char.ID, true), 2, now)
	expect(proto.SkillUseNoTarget)
	if a.bots[2].char.CurrentHP <= 1200 || s2.char.CurrentHP != 9000 {
		t.Fatal("heal affected the wrong character")
	}
}
