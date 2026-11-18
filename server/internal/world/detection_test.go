package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

// Detection (buff 4/93): every sweep ends the stealth of enemy users within
// ApplyRange of the caster, following the caster; allies are left alone.
func TestDetectionRevealsHiddenEnemiesInRangeAndFollowsTheCaster(t *testing.T) {
	s, _ := potentialSession(t)
	foe := detectionUser(t, s, 3, true)
	ally := detectionUser(t, s, 3, false)
	far := detectionUser(t, s, 40, true)
	for _, u := range []*Session{foe, ally, far} {
		if _, ok := u.addEffect(db.MustDefs().Skill(63, 1), 0, u.char.ID); !ok || !u.player.hidden.Load() {
			t.Fatal("stealth was not applied")
		}
	}
	def := db.MustDefs().Skill(43, 1)
	if def == nil || def.TypeEffect != typeEffectBuff || def.TypeDetail != detailDetection || def.ApplyRange != 5 {
		t.Fatalf("Detection 1 definition: %+v", def)
	}
	if _, ok := s.addEffect(def, 0, s.char.ID); !ok {
		t.Fatal("Detection was not applied")
	}
	s.tickDetection(time.Now())
	for _, u := range []*Session{foe, ally, far} {
		u.tickReveal()
	}
	if foe.player.hidden.Load() || foe.hasBuff(63) {
		t.Fatal("the enemy in range stayed hidden")
	}
	if !ally.player.hidden.Load() || !far.player.hidden.Load() {
		t.Fatal("an ally or an enemy out of range was revealed")
	}

	// The area moves with the caster.
	s.char.PosX += 37
	s.player.mu.Lock()
	s.player.character.PosX += 37
	s.player.mu.Unlock()
	s.tickDetection(time.Now().Add(2 * time.Second))
	far.tickReveal()
	if far.player.hidden.Load() {
		t.Fatal("the sweep did not follow the caster")
	}
}

func detectionUser(t *testing.T, s *Session, dx float32, enemy bool) *Session {
	t.Helper()
	u, _ := gmWireSession(t, *s.char, false)
	u.faction = s.faction
	if enemy {
		u.faction = FactionDark
		if s.faction == FactionDark {
			u.faction = FactionLight
		}
	}
	u.char.ID += uint32(1000 + int(dx)*3)
	if !enemy {
		u.char.ID++
	}
	u.char.Name = "Det" + string(rune('A'+int(dx)%20)) + map[bool]string{true: "E", false: "A"}[enemy]
	u.skillCooldowns, u.itemCooldowns = map[uint16]time.Time{}, map[uint8]time.Time{}
	u.char.PosX += dx
	u.player = RegisterPlayer(*u.char, nil, false, u.faction, u.out, u.logger)
	t.Cleanup(u.player.Unregister)
	return u
}
