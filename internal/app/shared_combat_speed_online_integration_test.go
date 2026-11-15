//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"os"
	"testing"
	"time"
)

func TestSharedCombatSpeedAgainstIsolatedGX(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_SHARED_SPEED") != "1" {
		t.Skip("requires isolated Ext. Fast opposing-faction fixtures")
	}
	st, wire, send, wait := testworld.Enter(t)
	other, _, _, waitOther := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	wait(func() bool {
		return st.Self.Speed != nil && st.Self.Speed.Attack == 9 && st.Entities[other.SelfID] != nil
	})
	waitOther(func() bool { return other.Entities[st.SelfID] != nil })
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	raw, err := arc.ReadFile("Character/Skill.SData")
	if err != nil {
		t.Fatal(err)
	}
	defs, err := sdata.ParseSkills(raw)
	if err != nil {
		t.Fatal(err)
	}
	a := inputApp()
	a.state, a.world, a.skillDefs = st, wire, defs
	started := time.Now()
	var stamps []time.Time
	var kinds []proto.Opcode
	var ids []uint16
	pump := func(d time.Duration) {
		t.Helper()
		end := time.Now().Add(d)
		for time.Now().Before(end) {
			a.clock = time.Since(started).Seconds()
			select {
			case p, ok := <-wire.Packets:
				if !ok {
					t.Fatal("closed shared-combat connection")
				}
				event, err := st.Apply(p)
				if err != nil {
					t.Fatal(err)
				}
				a.confirmedSkills(event.Op)
				a.confirmedAnimation(p)
				accepted := event.Op == proto.CharacterUserAutoAttack && st.LastAttack.Source == st.SelfID
				skill := uint16(0)
				if (event.Op == proto.SkillUse || event.Op == proto.SkillUseNoTarget) && st.LastSkill.Source == st.SelfID && st.LastSkill.Result <= 4 {
					accepted = true
					skill = st.LastSkill.ID
				}
				if accepted {
					stamps = append(stamps, time.Now())
					kinds = append(kinds, event.Op)
					ids = append(ids, skill)
				}
			default:
				time.Sleep(time.Millisecond)
			}
			a.updateQueuedSkill(&platform.Input{})
		}
	}
	next := func(count int) {
		t.Helper()
		end := time.Now().Add(4 * time.Second)
		for len(stamps) < count && time.Now().Before(end) {
			pump(5 * time.Millisecond)
		}
		if len(stamps) != count {
			t.Fatal("missing shared-cadence action", len(stamps), count, st.LastSkill)
		}
	}
	send(proto.UserAttackPacket(other.SelfID))
	next(1)
	if kinds[0] != proto.CharacterUserAutoAttack {
		t.Fatal("first action was not basic")
	}
	if a.prepareSkillMovement(proto.UseSkillPacket(42, st.SelfID, true)) || a.skills.queuedSkill == nil {
		t.Fatal("spell not buffered behind basic recovery")
	}
	for range 40 {
		send(proto.UseSkillPacket(43, st.SelfID, true))
	}
	pump(150 * time.Millisecond)
	if len(stamps) != 1 {
		t.Fatal("packet flood bypassed recovery")
	}
	next(2)
	if ids[1] != 30 {
		t.Fatal("queued first skill did not release", ids)
	}
	if a.prepareSkillMovement(proto.UseSkillPacket(43, st.SelfID, true)) || a.skills.queuedSkill == nil {
		t.Fatal("second combo skill was not buffered")
	}
	next(3)
	if ids[2] != 31 {
		t.Fatal("queued second skill did not release", ids)
	}
	send(proto.UserAttackPacket(other.SelfID))
	next(4)
	send(proto.StopAttackPacket())
	if kinds[3] != proto.CharacterUserAutoAttack {
		t.Fatal("basic did not follow skill")
	}
	for i := 1; i < len(stamps); i++ {
		gap := stamps[i].Sub(stamps[i-1])
		if gap < 940*time.Millisecond || gap > 1500*time.Millisecond {
			t.Fatal("Ext. Fast shared interval mismatch", i, gap)
		}
	}
	t.Log("encrypted basic -> buffered spell -> buffered spell -> basic: shared 1000ms cap, raw packet flood refused")
}
