//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"os"
	"runtime"
	"testing"
)

func TestAttackPauseStopsPredictedMovementGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 128, Height: 128, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err = a.Preview("world-stream"); err != nil {
		t.Fatal(err)
	}
	v := a.wv
	a.clock = 10
	v.attackMoveUntil = 10.5
	v.sendT = 1000
	v.moving = false
	v.motion = 0
	before := v.pos
	a.updateWorld(&platform.Input{Down: "W"}, .1)
	if v.pos[0] != before[0] || v.pos[2] != before[2] {
		t.Fatal("prediction moved during attack recovery")
	}
	v.self.PlayAction(scene.AttackAction(1, 0), false)
	v.attackPose = true
	v.attackActionID = v.self.ActionID()
	a.clock = 10.6
	v.moving = true
	v.motion = 1
	v.sendT = 1000
	a.updateWorld(&platform.Input{Down: "W"}, 1.0/60)
	if v.pos[0] == before[0] && v.pos[2] == before[2] {
		t.Fatal("movement did not resume")
	}
	if v.self.ActionActive() {
		t.Fatal("attack animation continued while running")
	}
	// A previous attack must never cancel an unrelated later jump action.
	v.attackPose = true
	v.startJumpAnimation()
	v.beginJump()
	v.moving, v.motion, v.sendT = true, 0x84, 1000
	a.updateWorld(&platform.Input{Down: "W"}, 1.0/60)
	if !v.self.PlayingAction(8) {
		t.Fatal("stale attack recovery canceled a later jump")
	}

	// A reply to a ground attack or spell can arrive after local takeoff.
	action := v.self.ActionID()
	a.state.LastCast = proto.SkillCast{Source: a.state.SelfID, ID: 71, Level: 1}
	a.confirmedAnimation(proto.NewWriter(proto.SkillCastStarted).Bytes())
	a.state.LastAttack = proto.AttackResult{Source: a.state.SelfID}
	a.confirmedAnimation(proto.NewWriter(proto.CharacterMobAutoAttack).Bytes())
	if v.self.PlayCast(85, 1) || v.self.PlayAction(8, false) {
		t.Fatal("another cast or jump could replace the jump clip")
	}
	v.moving, v.motion, v.sendT = false, 0x84, 1000
	a.updateWorld(&platform.Input{}, 1.0/60)
	if v.self.ActionID() != action || !v.self.PlayingAction(8) || !v.jumping || a.attackMovementAllowed() {
		t.Fatal("late combat reply cut the jump or enabled an attack")
	}
	a.clock = v.attackMoveUntil + .01
	v.moving, v.motion, v.sendT = true, 0x84, 1000
	a.updateWorld(&platform.Input{Down: "W"}, 1.0/60)
	if v.self.ActionID() != action || !v.self.PlayingAction(8) {
		t.Fatal("late attack identified the jump as an interruptible attack pose")
	}

}
