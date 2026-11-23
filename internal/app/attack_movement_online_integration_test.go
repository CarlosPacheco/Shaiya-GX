//go:build integration

package app

import (
	"os"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
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
	// A direct request in a standing jump must fail. The client's normal
	// attack entry point holds the request until movement and pose both end.
	send(proto.MotionPacket(proto.MotionJump))
	pump(50 * time.Millisecond)
	before := attacks
	send(proto.UserAttackPacket(b.SelfID))
	client := inputApp()
	client.state, client.world = a, wire
	weapon := a.Inventory[[2]uint8{0, 5}]
	client.items = map[sdata.ItemKey]*sdata.Item{{Type: weapon.Type, TypeID: weapon.TypeID}: {Range: 4}}
	client.wv.pos, client.wv.jumpActive = render.V3{x, y, z}, true
	a.Target = b.SelfID
	if !client.attackTarget() || client.wv.landingAction.target != b.SelfID {
		t.Fatal("client did not hold the attack order")
	}
	client.updateLandingAction(&platform.Input{Space: true})
	pump(350 * time.Millisecond)
	if attacks != before || !a.LastPlayerCorrection.Jumping {
		t.Fatal("stationary jump failed to become airborne or dealt a hit")
	}
	pump(450 * time.Millisecond)
	if a.LastPlayerCorrection.Jumping || attacks != before {
		t.Fatal("World did not land or retained the direct airborne request")
	}
	client.wv.jumpActive, client.clock = false, 3
	client.updateLandingAction(&platform.Input{Space: true})
	pump(150 * time.Millisecond)
	if attacks != before+1 || client.wv.landingAction.target != 0 {
		t.Fatal("client did not attack once on landing")
	}
	send(proto.StopAttackPacket())
	pump(1500 * time.Millisecond)
	send(proto.MotionPacket(proto.MotionJump))
	pump(30 * time.Millisecond)
	before = attacks
	client.wv.jumpActive, client.wv.jumpMoved = true, true
	if !client.attackTarget() || client.wv.landingAction.target != b.SelfID {
		t.Fatal("client did not hold the attack order during a moving jump")
	}
	for i := 0; i < 5; i++ {
		x += .05
		send(proto.MovePacket(4|proto.MotionFlag80, 0, x, y, z))
		send(proto.UserAttackPacket(b.SelfID))
		client.updateLandingAction(&platform.Input{Down: "W"})
		pump(40 * time.Millisecond)
	}
	pump(900 * time.Millisecond)
	if attacks != before {
		t.Fatal("moving jumping attacker dealt a hit")
	}
	client.wv.jumpActive, client.wv.jumpMoved, client.clock = false, false, 6
	client.updateLandingAction(&platform.Input{})
	pump(150 * time.Millisecond)
	if attacks != before+1 || client.wv.landingAction.target != 0 {
		t.Fatal("client did not attack once after the moving jump landed")
	}
	send(proto.StopAttackPacket())
	pump(1500 * time.Millisecond)
	// Movement by the victim is never immunity: attacker is now grounded.
	sendB(proto.MotionPacket(proto.MotionJump))
	sendB(proto.MovePacket(4|proto.MotionFlag80, 0, b.Self.X+.1, b.Self.Y, b.Self.Z))
	pump(50 * time.Millisecond)
	hit()
	t.Log("encrypted PvP: ground recovery enforced, all airborne attacks denied, standing and moving jumps execute queued attacks once on landing")
}
