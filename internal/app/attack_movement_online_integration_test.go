//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"os"
	"testing"
	"time"
)

func TestAttackMovementAgainstIsolatedGX(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_ATTACK_MOVEMENT") != "1" {
		t.Skip("requires isolated opposing-faction fixtures")
	}
	a, wire, send, wait := testworld.Enter(t)
	b, _, sendB, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	wait(func() bool { return a.Entities[b.SelfID] != nil && a.Self.Speed != nil })
	waitB(func() bool { return b.Entities[a.SelfID] != nil })
	attacks := 0
	pump := func(d time.Duration) {
		t.Helper()
		timer := time.NewTimer(d)
		defer timer.Stop()
		for {
			select {
			case p, ok := <-wire.Packets:
				if !ok {
					t.Fatal("connection closed")
				}
				if _, err := a.Apply(p); err != nil {
					t.Fatal(err)
				}
				if proto.Op(p) == proto.CharacterUserAutoAttack && a.LastAttack.Source == a.SelfID {
					attacks++
				}
			case <-timer.C:
				return
			}
		}
	}
	hit := func() {
		t.Helper()
		old := attacks
		send(proto.UserAttackPacket(b.SelfID))
		end := time.Now().Add(time.Second)
		for attacks == old && time.Now().Before(end) {
			pump(10 * time.Millisecond)
		}
		if attacks == old {
			t.Fatal("allowed attack was not resolved")
		}
		send(proto.StopAttackPacket())
	}
	x, y, z := a.Self.X, a.Self.Y, a.Self.Z
	send(proto.MovePacket(1, 0, x+.1, y, z))
	x += .1
	hit()
	// A forged displacement cannot move the attacker during the 800ms stop.
	send(proto.MovePacket(1, 0, x+1, y, z))
	pump(100 * time.Millisecond)
	if c := a.LastPlayerCorrection; c.ID != a.SelfID || c.X > x+.02 || c.X < x-.02 {
		t.Fatal("attack pause did not reject movement", c, x)
	}
	pump(1500 * time.Millisecond)
	// A vertical jump alone remains a valid attacking stance.
	send(proto.MotionPacket(proto.MotionJump))
	pump(50 * time.Millisecond)
	hit()
	pump(350 * time.Millisecond)
	if !a.LastPlayerCorrection.Jumping {
		t.Fatal("stationary jump was not physically airborne")
	}
	pump(1300 * time.Millisecond)
	send(proto.MotionPacket(proto.MotionJump))
	pump(30 * time.Millisecond)
	before := attacks
	for i := 0; i < 5; i++ {
		x += .05
		send(proto.MovePacket(4|proto.MotionFlag80, 0, x, y, z))
		send(proto.UserAttackPacket(b.SelfID))
		pump(40 * time.Millisecond)
	}
	if attacks != before {
		t.Fatal("moving jumping attacker dealt a hit")
	}
	send(proto.StopAttackPacket())
	pump(1100 * time.Millisecond)
	// Movement by the victim is never immunity: attacker is now grounded.
	sendB(proto.MotionPacket(proto.MotionJump))
	sendB(proto.MovePacket(4|proto.MotionFlag80, 0, b.Self.X+.1, b.Self.Y, b.Self.Z))
	pump(50 * time.Millisecond)
	hit()
	t.Log("encrypted PvP: 800ms movement stop enforced, vertical-jump attack allowed, moving-jump attacker denied, moving-jump victim remains hittable")
}
