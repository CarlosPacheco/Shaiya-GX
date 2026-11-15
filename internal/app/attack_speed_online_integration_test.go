//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"os"
	"testing"
	"time"
)

func TestAttackSpeedAgainstIsolatedGX(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_ATTACK_SPEED") != "1" {
		t.Skip("requires disposable opposing-faction speed fixtures")
	}
	a, wire, send, wait := testworld.Enter(t)
	wait(func() bool { return a.Self.Speed != nil && a.Self.Speed.Attack == 7 })
	b, _, _, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	waitB(func() bool { e := b.Entities[a.SelfID]; return e != nil && e.Speed != nil && e.Speed.Attack == 7 })
	wait(func() bool { return a.Entities[b.SelfID] != nil })
	cadence := func(ms int) {
		t.Helper()
		send(proto.UserAttackPacket(b.SelfID))
		var previous time.Time
		received := 0
		deadline := time.NewTimer(10 * time.Second)
		defer deadline.Stop()
		for received < 3 {
			select {
			case p, ok := <-wire.Packets:
				if !ok {
					t.Fatal("attack connection closed")
				}
				if _, err := a.Apply(p); err != nil {
					t.Fatal(err)
				}
				if proto.Op(p) != proto.CharacterUserAutoAttack || a.LastAttack.Source != a.SelfID {
					continue
				}
				now := time.Now()
				if !previous.IsZero() {
					delta := now.Sub(previous)
					if delta < time.Duration(ms-70)*time.Millisecond || delta > time.Duration(ms+500)*time.Millisecond {
						t.Fatalf("tier %d cadence %v expected %dms", a.Self.Speed.Attack, delta, ms)
					}
				}
				previous = now
				received++
				// Restarting and flooding requests cannot restart the server cooldown.
				send(proto.StopAttackPacket())
				for range 40 {
					send(proto.UserAttackPacket(b.SelfID))
				}
			case <-deadline.C:
				t.Fatal("missing authoritative attacks", received)
			}
		}
		send(proto.StopAttackPacket())
	}
	cadence(1500) // sword 5 + Flash 1 + learned one-handed mastery 1.
	send(proto.ItemMovePacket(1, 20, 0, 9))
	wait(func() bool { return a.Self.Speed.Attack == 8 })
	waitB(func() bool { return b.Entities[a.SelfID].Speed.Attack == 8 })
	cadence(1250) // Native Flash Ring AttackTime +1.
	send(proto.UseSkillPacket(41, a.SelfID, true))
	wait(func() bool { return a.Self.Speed.Attack == 9 })
	waitB(func() bool { return b.Entities[a.SelfID].Speed.Attack == 9 })
	cadence(1000) // Active +2 buff clamps to index 9.
	send(proto.CancelBuffPacket(129))
	wait(func() bool { return a.Self.Speed.Attack == 8 })
	send(proto.ItemMovePacket(0, 9, 1, 20))
	wait(func() bool { return a.Self.Speed.Attack == 7 })
	for _, command := range []string{"/char on", "/char off", "/attack on", "/attack off"} {
		n := len(a.Chat)
		send(proto.ChatPacket(proto.ChannelNormal, false, command))
		wait(func() bool { return len(a.Chat) > n })
		if a.Chat[len(a.Chat)-1].Text != "GM: command unavailable." || a.Self.Concealed {
			t.Fatal("retired GM control changed state")
		}
	}
	t.Log("encrypted PvP: native weapon + Flash + mastery + ring + active buff, viewer updates and 1500/1250/1000ms cadence survive stop/start packet floods")
}
