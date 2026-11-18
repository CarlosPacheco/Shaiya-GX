package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func cutBuff(id uint16, seconds int16) *db.SkillDef {
	return &db.SkillDef{SkillID: id, SkillLevel: 1, TypeEffect: typeEffectBuff, TypeDetail: 82, KeepSeconds: 30,
		AbilityType: [3]uint8{abilityCooldownCut}, AbilityValue: [3]int16{seconds}}
}

func remaining(s *Session, id uint16) time.Duration {
	at, ok := s.skillCooldowns[id]
	if !ok {
		return 0
	}
	return time.Until(at)
}

// A cooldown reduction takes seconds off spells already cooling down at once,
// and off the reuse of spells cast while it lasts. Potentials are not reduced.
func TestCooldownReductionRunningAndNewCooldowns(t *testing.T) {
	s, wire := potentialSession(t)
	const slow, quick, passive = uint16(61101), uint16(61102), uint16(24)
	s.skills = []db.CharacterSkill{{SkillID: slow, Level: 1, Number: 1}, {SkillID: quick, Level: 1, Number: 2}, {SkillID: passive, Level: 3, Number: 3}}
	defs := db.MustDefs()
	defs.Skills[db.SkillKey{SkillID: slow, Level: 1}] = &db.SkillDef{SkillID: slow, SkillLevel: 1, ResetTimeSeconds: 60}
	defs.Skills[db.SkillKey{SkillID: quick, Level: 1}] = &db.SkillDef{SkillID: quick, SkillLevel: 1, ResetTimeSeconds: 20}
	now := time.Now()
	s.skillCooldowns[slow] = now.Add(50 * time.Second)
	s.skillCooldowns[quick] = now.Add(10 * time.Second)
	s.skillCooldowns[passive] = now.Add(100 * time.Second)

	// 60 s spell with 50 s left and a 30 s reduction: 20 s left at once.
	if _, ok := s.addEffect(cutBuff(61200, 30), 0, s.char.ID); !ok {
		t.Fatal("reduction effect refused")
	}
	if d := remaining(s, slow); d < 19*time.Second || d > 21*time.Second {
		t.Fatalf("slow spell has %v left, want about 20s", d)
	}
	if _, running := s.skillCooldowns[quick]; running {
		t.Fatal("a remainder smaller than the reduction must be ready at once")
	}
	if d := remaining(s, passive); d < 99*time.Second {
		t.Fatalf("a Potential reuse was reduced: %v", d)
	}
	told := map[uint16]proto.SkillCooldown{}
	for len(wire) > 0 {
		if p := <-wire; proto.Op(p) == proto.GXSkillCooldown {
			c, _ := proto.DecodeSkillCooldown(p)
			told[c.Skill] = c
		}
	}
	if c := told[slow]; c.Seconds < 19 || c.Seconds > 21 || c.Total != 0 {
		t.Fatalf("client told %+v, want about 20s and an unchanged total", c)
	}
	if c, ok := told[quick]; !ok || c.Seconds != 0 {
		t.Fatalf("client not told the quick spell is ready: %+v", c)
	}

	// Cast while the reduction lasts: 60 s becomes 30 s with a full bar of 30.
	s.startCooldown(db.CharacterSkill{SkillID: slow, Level: 1}, defs.Skill(slow, 1))
	if d := remaining(s, slow); d < 29*time.Second || d > 31*time.Second {
		t.Fatalf("new reuse %v, want about 30s", d)
	}
	// A 20 s spell with a 30 s reduction is free to spam.
	s.startCooldown(db.CharacterSkill{SkillID: quick, Level: 1}, defs.Skill(quick, 1))
	if _, running := s.skillCooldowns[quick]; running || !s.skillReady(quick) {
		t.Fatal("a spell shorter than the reduction must have no reuse")
	}
	for len(wire) > 0 {
		if p := <-wire; proto.Op(p) == proto.GXSkillCooldown {
			c, _ := proto.DecodeSkillCooldown(p)
			told[c.Skill] = c
		}
	}
	if c := told[slow]; c.Seconds < 29 || c.Seconds > 31 || c.Total != c.Seconds {
		t.Fatalf("new timer announced as %+v", c)
	}

	// Once the effect ends spells cool down normally again.
	s.removeEffects(func(b *ActiveBuff) bool { return b.SkillID == 61200 }, "test expiry")
	s.startCooldown(db.CharacterSkill{SkillID: slow, Level: 1}, defs.Skill(slow, 1))
	if d := remaining(s, slow); d < 59*time.Second {
		t.Fatalf("reuse %v after the effect ended, want 60s", d)
	}
}

// Potential Motive (ability 19, value 500) clears every ordinary cooldown
// when it triggers, keeps its own 180 s reuse and leaves other Potentials.
func TestPotentialMotiveReducesCooldownsAndKeepsItsOwn(t *testing.T) {
	s, _ := potentialSession(t)
	const spell = uint16(61103)
	s.skills = []db.CharacterSkill{{SkillID: 28, Level: 1, Number: 1}, {SkillID: spell, Level: 1, Number: 2}, {SkillID: 24, Level: 1, Number: 3}}
	db.MustDefs().Skills[db.SkillKey{SkillID: spell, Level: 1}] = &db.SkillDef{SkillID: spell, SkillLevel: 1, ResetTimeSeconds: 300}
	now := time.Now()
	s.skillCooldowns[spell] = now.Add(250 * time.Second)
	s.skillCooldowns[24] = now.Add(150 * time.Second)
	s.char.CurrentHP = s.stats().MaxHP * 5 / 100 // below Motive's 10 percent, and Power's
	s.tickPotentials(now)
	if !s.hasBuff(28) {
		t.Fatal("Motive did not trigger")
	}
	if !s.skillReady(spell) {
		t.Fatal("Motive did not clear the running spell cooldown")
	}
	if d := remaining(s, 28); d < 179*time.Second {
		t.Fatalf("Motive reduced its own reuse: %v", d)
	}
	if d := remaining(s, 24); d < 149*time.Second {
		t.Fatalf("another Potential's reuse was reduced: %v", d)
	}
}

// A GM with invulnerability on still triggers a Potential.
func TestPotentialTriggersWhileInvulnerable(t *testing.T) {
	s, _ := potentialSession(t)
	s.isAdmin, s.gmImmune = true, true
	s.skills = []db.CharacterSkill{{SkillID: 24, Level: 3, Number: 1}}
	s.char.CurrentHP = s.stats().MaxHP * 5 / 100
	s.tickPotentials(time.Now())
	if !s.hasBuff(24) || s.skillReady(24) {
		t.Fatal("the Potential did not trigger for an invulnerable character")
	}
}
