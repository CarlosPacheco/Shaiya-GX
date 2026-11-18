package db

import "testing"

// Migration 7 removes dead skill levels, the chain relation, Aggravation's
// time damage and detail 25, and keeps every working level.
func TestSkillCatalogCleanup(t *testing.T) {
	p := sqliteFixture(t)
	count := func(query string, args ...any) int {
		var n int
		if err := p.QueryRow(query, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	for _, level := range [][2]int{{174, 3}, {302, 3}, {309, 3}, {320, 2}, {320, 3}, {201, 2}, {201, 3}, {268, 3}, {15, 3}, {176, 3}, {319, 3}, {125, 2}, {180, 3}} {
		if count(`SELECT COUNT(*) FROM PS_GameDefs_Skills WHERE SkillID=? AND SkillLevel=?`, level[0], level[1]) != 0 {
			t.Fatalf("dead level %v survived", level)
		}
	}
	for _, level := range [][2]int{{174, 1}, {174, 2}, {302, 1}, {302, 2}, {201, 1}, {268, 2}, {320, 1}, {150, 3}, {57, 3}, {176, 1}, {15, 2}, {125, 1}, {250, 2}, {274, 3}, {275, 3}, {280, 3}, {285, 3}} { // the last five are item-only levels
		if count(`SELECT COUNT(*) FROM PS_GameDefs_Skills WHERE SkillID=? AND SkillLevel=?`, level[0], level[1]) != 1 {
			t.Fatalf("working level %v was removed", level)
		}
	}
	if n := count(`SELECT COUNT(*) FROM PS_GameDefs_Skills WHERE SkillID IN (61,64,79,94,104,135,145,150,161,302,311,312,313) AND SkillLevel<100 AND PrevSkillID=0`); n != 0 {
		t.Fatalf("%d levels lost the skill that must be cooling down (migration 9)", n)
	}
	if n := count(`SELECT COUNT(*) FROM PS_GameDefs_Skills WHERE SkillID IN (309,310) AND SkillLevel<100 AND PrevSkillID=63`); n == 0 {
		t.Fatal("the Stealth requirement of Shadow Attack/Sleep must stay")
	}
	if n := count(`SELECT COUNT(*) FROM PS_GameDefs_Skills WHERE SkillID=150 AND (TimeDamageHP<>0 OR TimeDamageSP<>0 OR TimeDamageMP<>0)`); n != 0 {
		t.Fatal("Aggravation kept its time damage")
	}
	if n := count(`SELECT COUNT(*) FROM PS_GameDefs_Skills WHERE TypeDetail=25 AND TypeEffect=1 AND SkillLevel<100`); n != 0 {
		t.Fatalf("%d attack levels still use detail 25", n)
	}
	// The five former detail 25 skills keep their damage fields.
	for _, id := range []int{57, 115, 150, 174, 314} {
		if count(`SELECT COUNT(*) FROM PS_GameDefs_Skills WHERE SkillID=? AND TypeDetail=21 AND TypeEffect=1`, id) == 0 {
			t.Fatalf("skill %d was not moved to detail 21", id)
		}
	}
	if count(`SELECT COUNT(*) FROM PS_GameDefs_Skills WHERE SkillID=57 AND SkillLevel=3 AND AddDamageHP=298`) != 1 {
		t.Fatal("Double Damage lost its extra damage")
	}
}
