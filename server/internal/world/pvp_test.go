package world

import "testing"

func TestUserDamageFormula(t *testing.T) {
	att := CharacterStats{Level: 1, Attack: [3]int32{500, 500, 500}}
	def := CharacterStats{Level: 60, Luc: 500, Guard: [3]int32{100, 150, 100}}
	for kind, normal := range [3]int32{400, 350, 400} {
		if got, crit := userDamage(&att, &def, kind); got != normal || crit {
			t.Fatalf("kind %d normal = %d, crit = %t", kind, got, crit)
		}
		att.Critical = fullChance + int64(def.Luc) // the target's LUC takes 0.01 points per point off
		if got, crit := userDamage(&att, &def, kind); got != normal*3/2 || !crit {
			t.Fatalf("kind %d critical = %d, crit = %t", kind, got, crit)
		}
		att.Critical = 0
	}
	def.Guard[attackMelee] = 900
	if got, _ := userDamage(&att, &def, attackMelee); got != 1 {
		t.Fatalf("minimum damage = %d", got)
	}
}

func TestUserHitChanceIgnoresLevels(t *testing.T) {
	low := CharacterStats{Level: 1}
	high := CharacterStats{Level: 60}
	if !userHits(&low, &high, attackMelee) {
		t.Fatal("equal ratings must hit even across a 59-level gap")
	}
	high.Evasion[attackMelee] = fullChance
	if userHits(&low, &high, attackMelee) {
		t.Fatal("unopposed 100% evasion must dodge")
	}
	low.Accuracy[attackMelee] = fullChance
	if !userHits(&low, &high, attackMelee) {
		t.Fatal("accuracy cancels the same amount of evasion")
	}
}

// CUser::GetExp 0x004581b0: dead level x 50 x ((dead - killer + 1) x 0.2
// + 1), at least 1, 0 when the dead user is 10+ levels above.
func TestPvPExp(t *testing.T) {
	cases := []struct {
		dead, killer uint16
		want         uint32
	}{
		{40, 40, 2400}, // 2000 x 1.2
		{49, 40, 7350}, // diff 9: 2450 x 3
		{50, 40, 0},    // diff 10
		{20, 40, 1},    // a negative factor floors at 1
		{30, 35, 600},  // 1500 x 0.2
	}
	for _, c := range cases {
		got := pvpExp(c.dead, c.killer)
		diff := float64(int32(c.dead) - int32(c.killer))
		want := uint32(max(int64(float64(c.dead)*50*((diff+1)*0.2+1)), 1))
		if diff > 9 {
			want = 0
		}
		if got != want {
			t.Fatalf("pvpExp(%d, %d) = %d, want %d", c.dead, c.killer, got, want)
		}
	}
}

// CDamage::GetMax keeps the first entry on a tie.
func TestDamageLedgerTop(t *testing.T) {
	s := &Session{}
	s.addDamage(damageByMob, 7, 50)
	s.addDamage(damageByUser, 1, 30)
	s.addDamage(damageByUser, 1, 20)
	s.addDamage(damageByUser, 2, 10)
	top, ok := s.topDamage()
	if !ok || top.Type != damageByMob || top.ID != 7 {
		t.Fatalf("top %+v, want the mob (first of the 50s)", top)
	}
	s.addDamage(damageByUser, 2, 41)
	if top, _ = s.topDamage(); top.Type != damageByUser || top.ID != 2 || top.Amount != 51 {
		t.Fatalf("top %+v, want user 2 with 51", top)
	}
}
