//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"testing"
	"time"
)

func verifyNativeApproach(t *testing.T, a *App, pump func(func() bool)) {
	t.Helper()
	v, st := a.wv, a.state
	frame := func() {
		time.Sleep(time.Second / 60)
		// This fixture drives updateWorld directly instead of App.Update.
		// Advance the app clock so attack recovery expires normally.
		a.clock += 1.0 / 60
		a.updateWorld(&platform.Input{}, 1.0/60)
		once := false
		pump(func() bool {
			if once {
				return true
			}
			once = true
			return false
		})
	}
	walk := func(p render.V3) {
		t.Helper()
		v.destination, v.travel = p, true
		for i := 0; i < 900 && v.travel; i++ {
			frame()
		}
		if v.travel {
			t.Fatal("fixture walk could not reach destination", v.pos, p)
		}
	}

	send := func(p []byte) {
		t.Helper()
		if err := a.world.Send(p); err != nil {
			t.Fatal(err)
		}
	}
	// Drop one unit from the isolated runner's disposable stack.
	key := [2]uint8{1, 23}
	count := st.Inventory[key].Count
	if count < 1 {
		t.Fatal("missing disposable pickup stack")
	}
	before := map[uint32]bool{}
	for id := range st.GroundItems {
		before[id] = true
	}
	drop := proto.NewWriter(proto.RemoveItem)
	drop.U8(1)
	drop.U8(23)
	drop.U8(1)
	send(drop.Bytes())
	var loot uint32
	pump(func() bool {
		for id, item := range st.GroundItems {
			if !before[id] && item.Type == 25 && item.TypeID == 4 {
				loot = id
				return true
			}
		}
		return false
	})
	item := st.GroundItems[loot]
	walk(render.V3{item.X + 8, item.Y, item.Z})
	if !a.pickupNearest() || !v.travel || v.approachTarget != loot {
		t.Fatal("pickup action did not start approach")
	}
	for i := 0; i < 600 && v.approachTarget != 0; i++ {
		frame()
	}
	if v.approachTarget != 0 {
		t.Fatal("loot approach never reached pickup distance")
	}
	pump(func() bool { _, present := st.GroundItems[loot]; return !present && st.Inventory[key].Count == count })
	// Pick a real visible living mob and begin fifteen units away.
	// First monster area in original map-one svmap: (789..809,1663..1680).

	enemy := testworld.LivingMob(t, st, send, pump)
	walk(render.V3{enemy.X + 10, enemy.Y, enemy.Z})
	// Exercise native selection before the attack action starts its approach.
	st.Target = 0
	a.updateCombat(&platform.Input{Keys: []platform.Key{platform.KeyTarget}}, 1024, 734)
	first := st.Target
	if first == 0 {
		t.Fatal("native target key did not acquire an enemy")
	}
	a.updateCombat(&platform.Input{Keys: []platform.Key{platform.KeyTarget}}, 1024, 734)
	if st.Target != first {
		a.updateCombat(&platform.Input{Keys: []platform.Key{platform.KeyPreviousTarget}}, 1024, 734)
		if st.Target != first {
			t.Fatal("Backspace did not restore previous target")
		}
	}
	st.Target = 0
	if !a.autoAttackNearest() || v.approachTarget == 0 {
		t.Fatal("auto attack did not acquire and approach a nearby enemy")
	}
	target := st.Target
	for i := 0; i < 600 && v.approachTarget != 0; i++ {
		frame()
	}
	if v.approachTarget != 0 {
		t.Fatal("attack approach did not finish")
	}
	pump(func() bool { return st.LastAttack.Source == st.SelfID && st.LastAttack.Target == target })
	if len(a.combatNumbers) == 0 {
		t.Fatal("confirmed basic attack had no combat number")
	}
	send(proto.StopAttackPacket())
	// The selected-target action (F/double click) also starts approach itself.
	enemy = testworld.LivingMob(t, st, send, pump)
	st.Target = enemy.ID
	walk(render.V3{enemy.X + 10, enemy.Y, enemy.Z})
	v.facing = render.V3{0, 0, 1}
	if !a.attackTarget() || v.approachTarget != enemy.ID || v.facing[0] > -.99 {
		t.Fatal("selected-target attack neither turned nor approached", v.facing, v.approachTarget)
	}
	// Cancel that approach with manual movement; then cast through the real bar.
	a.updateActionApproach(&platform.Input{Down: "S"})
	send(proto.LearnSkillPacket(57, 1))
	pump(func() bool { return st.LastLearn.ID == 57 })
	if st.LastLearn.Result != 0 {
		t.Fatal("fighter could not learn Double Damage", st.LastLearn)
	}
	st.QuickSlots = []proto.QuickSlot{{Bar: 0, Slot: 0, Bag: 100, Number: uint16(st.LastLearn.Number)}}
	a.useAction(0)
	if v.approachTarget != enemy.ID || len(v.approachSkill) == 0 {
		t.Fatal("bar skill did not queue approach")
	}
	for i := 0; i < 600 && v.approachTarget != 0; i++ {
		frame()
	}
	if v.approachTarget != 0 {
		t.Fatal("targeted skill approach did not finish")
	}
	pump(func() bool { return st.LastSkill.ID == 57 && st.LastSkill.Target == enemy.ID })
	if st.LastSkill.Result > 3 {
		t.Fatal("approached skill rejected", st.LastSkill)
	}
	if a.skills.cooldowns[57] <= a.clock || a.actionPacket(0) != nil {
		t.Fatal("confirmed skill reuse did not block another bar request")
	}
	t.Log("selected-target attack turned immediately; bar skill approached, was confirmed by server, and blocked repeated use during its real cooldown")
	frame()
	t.Log("action approach: nearest loot reached and picked up after ACK; nearest enemy reached and attacked after server confirmation")
}
