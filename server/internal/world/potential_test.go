package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func potentialSession(t *testing.T) (*Session, <-chan []byte) {
	t.Helper()
	base, _ := lapisSession(t)
	s, wire := gmWireSession(t, *base.char, false)
	s.items = base.items
	s.skillCooldowns, s.itemCooldowns = map[uint16]time.Time{}, map[uint8]time.Time{}
	s.player = RegisterPlayer(*s.char, nil, false, s.faction, s.out, s.logger)
	t.Cleanup(s.player.Unregister)
	return s, wire
}

// The Potentials are passive skills: below their own HP margin they apply
// themselves as a 30 second buff and start a three minute reuse; they cannot
// be cast.
func TestPotentialTriggersBelowItsMarginOncePerReuse(t *testing.T) {
	s, wire := potentialSession(t)
	// Power Lv3 triggers below 20 percent, Motive Lv1 below 10 percent.
	s.skills = []db.CharacterSkill{{SkillID: 24, Level: 3, Number: 1}, {SkillID: 28, Level: 1, Number: 2}}
	def := db.MustDefs().Skill(24, 3)
	if def == nil || !isPotential(def) || def.KeepSeconds != 30 || def.ResetTimeSeconds != 180 || def.LimitHP != 20 {
		t.Fatalf("potential definition not normalized: %+v", def)
	}
	before := s.stats()
	now := time.Now()

	s.char.CurrentHP = before.MaxHP
	s.tickPotentials(now)
	if s.hasBuff(24) || len(s.skillCooldowns) != 0 {
		t.Fatal("triggered at full health")
	}
	s.char.CurrentHP = before.MaxHP * 20 / 100 // exactly on the margin is not below it
	s.tickPotentials(now)
	if s.hasBuff(24) {
		t.Fatal("triggered on the margin instead of below it")
	}

	s.char.CurrentHP = before.MaxHP * 19 / 100
	s.tickPotentials(now)
	if !s.hasBuff(24) || s.hasBuff(28) {
		t.Fatal("Power must trigger at 19 percent and Motive must wait for 10")
	}
	if at := s.skillCooldowns[24]; at.Before(now.Add(179*time.Second)) || at.After(now.Add(181*time.Second)) {
		t.Fatalf("reuse is %v, want three minutes", time.Until(at))
	}
	for _, b := range s.buffs {
		if b.SkillID == 24 {
			if d := time.Until(b.EndsAt); d < 29*time.Second || d > 31*time.Second {
				t.Fatalf("buff lasts %v, want 30 seconds", d)
			}
		}
	}
	after := s.stats()
	if after.Attack[0] <= before.Attack[0] || after.AttackSpeed != before.AttackSpeed+3 {
		t.Fatalf("abilities not applied: attack %d -> %d, speed %d -> %d", before.Attack[0], after.Attack[0], before.AttackSpeed, after.AttackSpeed)
	}
	waitForReuse := time.NewTimer(time.Second)
	defer waitForReuse.Stop()
reuse:
	for {
		select {
		case p, ok := <-wire:
			if !ok {
				t.Fatal("the wire closed before the reuse update")
			}
			if proto.Op(p) == proto.GXSkillCooldown {
				c, err := proto.DecodeSkillCooldown(p)
				if err == nil && c.Skill == 24 && c.Seconds == 180 && c.Total == 180 {
					break reuse
				}
			}
		case <-waitForReuse.C:
			t.Fatal("the client was not told about the reuse")
		}
	}

	// Expired buff, reuse still running: no new trigger.
	s.removeEffects(func(b *ActiveBuff) bool { return b.SkillID == 24 }, "test expiry")
	s.tickPotentials(now.Add(time.Minute))
	if s.hasBuff(24) {
		t.Fatal("triggered again during its reuse")
	}
	s.skillCooldowns[24] = now.Add(-time.Second)
	s.tickPotentials(now.Add(4 * time.Minute))
	if !s.hasBuff(24) {
		t.Fatal("did not trigger after its reuse ended")
	}

	// Below 10 percent Motive triggers too (its cooldown reduction is tested apart).
	s.char.CurrentHP = before.MaxHP * 9 / 100
	s.tickPotentials(now)
	if !s.hasBuff(28) {
		t.Fatal("Motive did not trigger below its margin")
	}
}

func TestPotentialCannotBeCast(t *testing.T) {
	s, wire := potentialSession(t)
	s.skills = []db.CharacterSkill{{SkillID: 25, Level: 1, Number: 1}}
	s.resolveSkill(0, 1, 0x0511, nil)
	s.char.CurrentHP = s.stats().MaxHP
	if s.hasBuff(25) || len(s.skillCooldowns) != 0 || len(wire) != 0 {
		t.Fatal("a Potential was cast")
	}
}
