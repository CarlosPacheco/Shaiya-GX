package sdata

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestSoulBurnCatalogPatchIsScopedAndRepeatable(t *testing.T) {
	groups := make([][]fakeSlot, 126)
	groups[124] = []fakeSlot{{"Soul Burn Lv1", "Old tooltip", 1, 4, 3, 122, 0, 0},
		{"Soul Burn Lv2", "Old tooltip", 2, 4, 3, 122, 0, 0}, {"Soul BurnED", "Not Available", 3, 4, 3, 122, 0, 0}}
	groups[125] = []fakeSlot{{"Resurrection Lv1", "Allied corpse", 1, 4, 6, 121, 0, 0}}
	plain := buildCatalog(groups)
	patched, report, err := EnableSoulBurn(plain)
	if err != nil || len(report.SoulBurnLevels) != 3 {
		t.Fatalf("Soul Burn patch: %v, %v", report, err)
	}
	defs, err := ParseSkills(patched)
	if err != nil {
		t.Fatal(err)
	}
	d := defs[SkillKey{ID: 125, Level: 1}]
	if d.Effect != 6 || !d.IsSoulBurn() || d.Classes != ([6]uint8{0, 0, 0, 0, 0, 1}) || d.Points != 3 || d.SuccessType != 1 || d.SuccessValue != 100 || d.Description != soulBurnDescription {
		t.Fatalf("wrong active Soul Burn definition: %+v", d)
	}
	if _, ok := defs[SkillKey{ID: 125, Level: 2}]; ok {
		t.Fatal("obsolete Soul Burn rank survived")
	}
	_, before, err := readSkillCatalog(plain)
	if err != nil {
		t.Fatal(err)
	}
	_, after, err := readSkillCatalog(patched)
	if err != nil {
		t.Fatal(err)
	}
	for i, s := range before {
		if s.id != 125 && (!bytes.Equal(s.rec, after[i].rec) || !bytes.Equal(s.nameRaw, after[i].nameRaw) || !bytes.Equal(s.descRaw, after[i].descRaw)) {
			t.Fatalf("unrelated skill %d changed", s.id)
		}
	}
	again, report, err := EnableSoulBurn(patched)
	if err != nil || len(report.SoulBurnLevels) != 0 || !bytes.Equal(again, patched) {
		t.Fatalf("patch is not idempotent: %v, %v", report, err)
	}
}

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
		nil, nil, // IDs 3 and 4 are retired.
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
	if got := defs[SkillKey{ID: 5, Level: 1}].Description; got != "Deals damage only in Stealth mode." {
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
	if rep.ChainTexts != 4 || len(rep.PrevRestored) != 1 || len(rep.DeadLevels) != 1 || len(rep.TimeDamage) != 1 || len(rep.DetailMoved) != 2 || rep.ManaStaminaText != 1 || len(rep.FullyDeadSkills) != 1 || rep.FullyDeadSkills[0] != 6 {
		t.Fatalf("report %s", rep)
	}
	// Idempotent: a second pass changes nothing.
	again, rep2, err := normalizeSkillCatalog(out, nil)
	if err != nil || string(again) != string(out) || rep2.ChainTexts != 0 || len(rep2.DeadLevels) != 0 {
		t.Fatalf("second pass changed the catalog: %v %s", err, rep2)
	}
}

func TestRemoveRetiredPlayerSkillsKeepsNeighborIDsAndUntouchedBytes(t *testing.T) {
	groups := [][]fakeSlot{nil, nil,
		{{"Interpretation", "Original description", 1, 1, 0, 5, 0, 0}},
		{{"Backpack Upgrade", "Original description", 3, 1, 0, 2, 0, 0}},
		{{"Sharpen", "Keep this description unchanged", 1, 1, 0, 1, 0, 0}},
	}
	out, rep, err := RemoveRetiredPlayerSkills(buildCatalog(groups))
	if err != nil || len(rep.RetiredLevels) != 2 {
		t.Fatalf("retirement: %v %v", rep, err)
	}
	groups[2], groups[3] = nil, nil
	if string(out) != string(buildCatalog(groups)) {
		t.Fatal("the patch modified records outside the retired slots")
	}
	again, rep, err := RemoveRetiredPlayerSkills(out)
	if err != nil || len(rep.RetiredLevels) != 0 || string(again) != string(out) {
		t.Fatalf("second pass changed catalog: %v %v", rep, err)
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

// The previous-skill field is kept: the description names the skill that has
// to be cooling down, once.
func TestRequireCooldown(t *testing.T) {
	got := requireCooldown("Deals 37 damage and stuns for 4 seconds", "Halt Kick")
	want := "Deals 37 damage and stuns for 4 seconds. Can only be used while Halt Kick is cooling down."
	if got != want || requireCooldown(got, "Halt Kick") != got || requireCooldown("Deals damage.", "") != "Deals damage." {
		t.Fatalf("got %q", got)
	}
	if levelSuffix.ReplaceAllString("Stun Crash Lv1", "") != "Stun Crash" {
		t.Fatal("level suffix kept")
	}
}
