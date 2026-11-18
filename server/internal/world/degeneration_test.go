package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/movespeed"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

// Degeneration (3/90): the target becomes a creature for the skill's time. It
// moves at the lowest tier, cannot attack or cast, and damage ends the effect.
func TestDegenerationSlowsBlocksActionsAndBreaksOnDamage(t *testing.T) {
	s, _ := potentialSession(t)
	def := db.MustDefs().Skill(97, 3)
	if def == nil || def.TypeEffect != typeEffectDebuff || def.TypeDetail != detailDegeneration || def.KeepSeconds != 20 {
		t.Fatalf("Degeneration 3 definition: %+v", def)
	}
	normal := s.speeds()
	if s.cannotAct() || normal.move == movespeed.Blocked {
		t.Fatal("a normal user was blocked")
	}
	if _, ok := s.addEffect(def, 0, s.char.ID+1); !ok || !s.degenerated() {
		t.Fatal("the transformation was not applied")
	}
	got := s.speeds()
	if got.move != 0 || got.move >= normal.move {
		t.Fatalf("move tier %d, want the lowest (was %d)", got.move, normal.move)
	}
	if got.attack != 255 || !s.cannotAct() {
		t.Fatal("a creature must not attack or cast")
	}
	if s.actionBlocked() {
		t.Fatal("consumables and equipment use actionBlocked paths: it must stay false")
	}

	// Damage ends it, and the user acts and moves normally again.
	s.wake()
	if s.degenerated() || s.cannotAct() {
		t.Fatal("damage did not end the transformation")
	}
	if s.speeds() != normal {
		t.Fatalf("speeds after the transformation %+v, want %+v", s.speeds(), normal)
	}
}

func TestDegenerationBlocksSkillUse(t *testing.T) {
	s, _ := potentialSession(t)
	s.skills = []db.CharacterSkill{{SkillID: 29, Level: 1, Number: 1}}
	s.skillCooldowns = map[uint16]time.Time{}
	if _, ok := s.addEffect(db.MustDefs().Skill(97, 1), 0, s.char.ID+1); !ok {
		t.Fatal("not applied")
	}
	before := s.stats().CurrentMP
	s.resolveSkill(0, 1, 0x0517, nil)
	if s.stats().CurrentMP != before || len(s.skillCooldowns) != 0 {
		t.Fatal("a transformed user cast a spell")
	}
}

// The monster look is announced when the transformation starts and ends.
func TestDegenerationAnnouncesTheMonsterLook(t *testing.T) {
	s, wire := potentialSession(t)
	last := func() (proto.Transform, bool) {
		time.Sleep(50 * time.Millisecond) // the pipe is read by another goroutine
		var out proto.Transform
		found := false
		for len(wire) > 0 {
			if p := <-wire; proto.Op(p) == proto.GXTransform {
				v, err := proto.DecodeTransform(p)
				if err != nil {
					t.Fatal(err)
				}
				out, found = v, true
			}
		}
		return out, found
	}
	s.tickTransform()
	if _, ok := last(); ok {
		t.Fatal("a look was announced without a transformation")
	}
	if _, ok := s.addEffect(db.MustDefs().Skill(97, 2), 0, s.char.ID+1); !ok {
		t.Fatal("not applied")
	}
	s.tickTransform()
	if v, ok := last(); !ok || v.User != s.char.ID || v.Mob != 60 {
		t.Fatalf("announcement %+v %v, want monster 60", v, ok)
	}
	if s.player.transformMob.Load() != 60 {
		t.Fatal("the new viewers' state was not kept")
	}
	s.wake()
	s.tickTransform()
	if v, ok := last(); !ok || v.Mob != 0 {
		t.Fatalf("end announcement %+v %v", v, ok)
	}
}

// A selected user's HP reaches the client, once and again when it changes.
func TestSelectedUserHealthIsSentWhenItChanges(t *testing.T) {
	s, wire := potentialSession(t)
	s.effectWatchKind, s.effectWatchID = proto.TargetEffectUser, s.char.ID
	s.char.CurrentHP = 1000
	s.player.UpdateCharacter(*s.char)
	healths := func() []proto.TargetHealth {
		time.Sleep(50 * time.Millisecond) // the pipe is read by another goroutine
		var out []proto.TargetHealth
		for len(wire) > 0 {
			if p := <-wire; proto.Op(p) == proto.GXTargetHealth {
				v, err := proto.DecodeTargetHealth(p)
				if err != nil {
					t.Fatal(err)
				}
				out = append(out, v)
			}
		}
		return out
	}
	now := time.Now()
	s.tickTargetEffects(now)
	first := healths()
	if len(first) != 1 || first[0].User != s.char.ID || first[0].Max == 0 || first[0].HP != 1000 {
		t.Fatalf("first report %+v", first)
	}
	s.tickTargetEffects(now.Add(time.Second))
	if got := healths(); len(got) != 0 {
		t.Fatalf("an unchanged HP was repeated: %+v", got)
	}
	s.char.CurrentHP = 400
	s.player.UpdateCharacter(*s.char)
	s.tickTargetEffects(now.Add(2 * time.Second))
	if got := healths(); len(got) != 1 || got[0].HP != 400 {
		t.Fatalf("a changed HP was not reported: %+v", got)
	}
}

// A selected Game Master's HP is reported and follows the damage it takes.
func TestSelectedGameMasterHealthFollowsDamage(t *testing.T) {
	viewer, wire := potentialSession(t)
	gm, _ := gmWireSession(t, *viewer.char, true)
	gm.char.ID, gm.char.Name = viewer.char.ID+900, "SelectedGM"
	gm.skillCooldowns, gm.itemCooldowns = map[uint16]time.Time{}, map[uint8]time.Time{}
	gm.player = RegisterPlayer(*gm.char, nil, true, gm.faction, gm.out, gm.logger)
	t.Cleanup(gm.player.Unregister)
	gm.char.CurrentHP = 1000
	gm.syncPlayer()
	viewer.player.know(viewUser, gm.char.ID)
	viewer.effectWatchKind, viewer.effectWatchID = proto.TargetEffectUser, gm.char.ID
	read := func() []proto.TargetHealth {
		time.Sleep(50 * time.Millisecond)
		var out []proto.TargetHealth
		for len(wire) > 0 {
			if p := <-wire; proto.Op(p) == proto.GXTargetHealth {
				v, _ := proto.DecodeTargetHealth(p)
				out = append(out, v)
			}
		}
		return out
	}
	now := time.Now()
	viewer.tickTargetEffects(now)
	if got := read(); len(got) != 1 || got[0].User != gm.char.ID || got[0].HP != 1000 {
		t.Fatalf("the GM's HP was not reported: %+v", got)
	}
	gm.takeUserDamage(viewer.char.ID, 300)
	viewer.tickTargetEffects(now.Add(time.Second))
	if got := read(); len(got) != 1 || got[0].HP != 700 {
		t.Fatalf("the damage did not reach the bar: %+v (GM HP %d)", got, gm.char.CurrentHP)
	}
}

// The killer is told whom it killed, and a kill gives no experience.
func TestKillNoticeNamesTheVictim(t *testing.T) {
	killer, wire := potentialSession(t)
	victim, _ := gmWireSession(t, *killer.char, false)
	victim.char.ID, victim.char.Name = killer.char.ID+950, "KilledOne"
	victim.player = RegisterPlayer(*victim.char, nil, false, victim.faction, victim.out, victim.logger)
	t.Cleanup(victim.player.Unregister)
	victim.notifyKill(killer.player)
	time.Sleep(50 * time.Millisecond)
	found := ""
	for len(wire) > 0 {
		if p := <-wire; proto.Op(p) == proto.GXKillNotice {
			found, _ = proto.DecodeKillNotice(p)
		}
	}
	if found != "KilledOne" {
		t.Fatalf("kill notice %q", found)
	}
}
