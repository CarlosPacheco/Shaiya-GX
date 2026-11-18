package sdata

import (
	"encoding/binary"
	"strings"
	"testing"
)

// catalog builds a plain EP5 catalog: groups of nine slots.
type fakeSlot struct {
	name, desc string
	level      byte
	show       byte
	effect     byte
	detail     uint16
	prev       uint16
	timeDamage int32
}

func buildCatalog(groups [][]fakeSlot) []byte {
	out := binary.LittleEndian.AppendUint32(nil, uint32(len(groups)))
	for _, g := range groups {
		for i := 0; i < 9; i++ {
			s := fakeSlot{}
			if i < len(g) {
				s = g[i]
			}
			rec := make([]byte, 116)
			if s.level != 0 {
				rec[0], rec[19], rec[21] = s.level, s.show, s.effect
				binary.LittleEndian.PutUint16(rec[22:], s.detail)
				binary.LittleEndian.PutUint16(rec[skillPrevOffset:], s.prev)
				binary.LittleEndian.PutUint32(rec[skillTimeDamageHP:], uint32(s.timeDamage))
			}
			out = append(append(append(out, lengthPrefixed(s.name)...), lengthPrefixed(s.desc)...), rec...)
		}
	}
	return out
}

// Dead levels can still carry stale effect fields; only their text marks them.
func TestDeadLevelsAreRecognisedByTheirText(t *testing.T) {
	groups := [][]fakeSlot{{
		{"Strength Remedy Lv1", "Increases STR by 15 for 1 hour.", 1, 0, 4, 82, 0, 0},
		{"Strength Remedy Lv2", "Increases STR by 35 for 1 hour.", 2, 0, 4, 82, 0, 0},
		{"Strength Remedy Lv3", "Not Available.", 3, 0, 4, 82, 0, 0},
	}}
	groups = append(groups, []fakeSlot{{"Blessing", "Blessing of the Goddess", 1, 0, 4, 90, 0, 0}, {"Blessing", "Noble Blessing", 2, 0, 4, 90, 0, 0}, {"Blessing", "Non-existant skill", 3, 0, 4, 90, 0, 0}})
	out, rep, err := normalizeSkillCatalog(buildCatalog(groups), nil)
	if err != nil {
		t.Fatal(err)
	}
	defs, _ := ParseSkills(out)
	for _, key := range []SkillKey{{1, 3}, {2, 3}} {
		if _, ok := defs[key]; ok {
			t.Fatalf("dead level %v survived", key)
		}
	}
	if _, ok := defs[SkillKey{ID: 1, Level: 2}]; !ok || len(rep.DeadLevels) != 2 {
		t.Fatalf("working levels must stay: %s", rep)
	}
}

func TestNormalizeSkillCatalog(t *testing.T) {
	// Groups are numbered from one; pad with empty groups up to ids 115 and 150.
	var groups [][]fakeSlot
	groups = append(groups, [][]fakeSlot{
		{{"Fatal Hit Lv1", "Opening skill of chain skill that deals 10 extra damage plus your Attack Power but the following chain skill must be used in 5 seconds.", 1, 3, 3, 21, 0, 0},
			{"Fatal Hit Lv2", "Opening skill of chain skill that deals 10 extra damage plus your Attack Power but the following chain skill must be used in 8 seconds.", 2, 3, 3, 21, 0, 0},
			{"Fatal HitED", "Not Available.", 3, 0, 0, 0, 0, 0}},
		{{"Tetanus Lv1", "Chain skill that requires Fatal Hit; Deals initial damage but the following chain skill must be used in 3 seconds.", 1, 3, 3, 21, 1, 0}},
		{{"Shadow Attack Lv1", "Deals damage only in Stealth mode.", 1, 3, 3, 21, 63, 0}},
		{{"Blank", "Skill not available currently", 1, 0, 0, 0, 0, 0}}, // fully dead: left alone
	}...)
	for len(groups) < 114 {
		groups = append(groups, nil)
	}
	groups = append(groups, []fakeSlot{
		{"Mana Shot Lv1", "Deals 243 damage on mana plus initial damage. Only usable on players from enemy realm.", 1, 3, 1, 25, 0, 0}})
	for len(groups) < 149 {
		groups = append(groups, nil)
	}
	groups = append(groups, []fakeSlot{
		{"Aggravation Lv1", "Chain skill that requires Tetanus; Deals damage but the following chain skill must be used in 3 seconds.", 1, 3, 1, 25, 2, 6}})
	plain := buildCatalog(groups)

	out, rep, err := normalizeSkillCatalog(plain, nil)
	if err != nil {
		t.Fatal(err)
	}
	defs, err := ParseSkills(out)
	if err != nil {
		t.Fatal(err)
	}
	if d := defs[SkillKey{ID: 1, Level: 1}].Description; d != "Deals 10 extra damage plus your Attack Power." {
		t.Fatalf("opener text %q", d)
	}
	if _, ok := defs[SkillKey{ID: 1, Level: 3}]; ok {
		t.Fatal("the dead third level survived")
	}
	if _, ok := defs[SkillKey{ID: 1, Level: 2}]; !ok {
		t.Fatal("a working level was removed")
	}
	if d := defs[SkillKey{ID: 2, Level: 1}].Description; d != "Deals initial damage." {
		t.Fatalf("chain text %q", d)
	}
	if got := defs[SkillKey{ID: 3, Level: 1}].Description; got != "Deals damage only in Stealth mode." {
		t.Fatalf("a stealth requirement was rewritten: %q", got)
	}
	if got := defs[SkillKey{ID: 115, Level: 1}]; got.Detail != 21 || strings.Contains(got.Description, "Only usable") || got.Description != "Deals 243 damage on mana plus initial damage." {
		t.Fatalf("Mana Shot: %+v", got)
	}
	if got := defs[SkillKey{ID: 150, Level: 1}]; got.Detail != 21 || got.Description != "Deals damage." {
		t.Fatalf("Aggravation: %+v", got)
	}
	// Raw checks the parser does not expose.
	plainOut, _ := Decrypt(out)
	if len(plainOut) != len(out) {
		t.Fatal("output is not plain")
	}
	if rep.ChainTexts != 4 || len(rep.PrevCleared) != 2 || len(rep.DeadLevels) != 1 || len(rep.TimeDamage) != 1 || len(rep.DetailMoved) != 2 || rep.ManaStaminaText != 1 || len(rep.FullyDeadSkills) != 1 || rep.FullyDeadSkills[0] != 4 {
		t.Fatalf("report %s", rep)
	}
	// Idempotent: a second pass changes nothing.
	again, rep2, err := normalizeSkillCatalog(out, nil)
	if err != nil || string(again) != string(out) || rep2.ChainTexts != 0 || len(rep2.DeadLevels) != 0 {
		t.Fatalf("second pass changed the catalog: %v %s", err, rep2)
	}
}

// The text fixes apply once, are skipped when already done, and refuse a
// catalog whose text is not the expected one.
func TestApplyTextFixes(t *testing.T) {
	plain := buildCatalog([][]fakeSlot{{{"A Lv1", "Increases Attack Speed by 3 level.", 1, 1, 0, 8, 0, 0}}})
	fixes := []byte("1,1,Increases Attack Speed by 3 level.,Increases Attack Speed by 2 levels.\n")
	out, rep, err := normalizeSkillCatalog(plain, fixes)
	if err != nil || rep.TextFixes != 1 {
		t.Fatal(err, rep)
	}
	defs, _ := ParseSkills(out)
	if d := defs[SkillKey{ID: 1, Level: 1}].Description; d != "Increases Attack Speed by 2 levels." {
		t.Fatalf("text %q", d)
	}
	again, rep2, err := normalizeSkillCatalog(out, fixes)
	if err != nil || rep2.TextFixes != 0 || string(again) != string(out) {
		t.Fatal("a second pass changed the catalog", err)
	}
	if _, _, err := normalizeSkillCatalog(plain, []byte("1,1,Some other text.,New text.\n")); err == nil {
		t.Fatal("an unexpected description was overwritten")
	}
}
