package sdata

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// GX skill catalog cleanup. NormalizeSkillCatalog rewrites Character/Skill.SData
// without touching any byte it does not need to:
//   - dead skill levels (a placeholder such as "Hell LordED" / "Not Available.")
//     of skills that have working levels become the same empty slot as the
//     unused levels; skills with no working level at all are left alone;
//   - the previous-skill field means "that skill must be cooling down for this
//     one to be usable": the original values are restored and the description
//     says so instead of using chain wording;
//   - Mana Shot and Stamina Shot are ordinary spells: their "enemy players
//     only" sentence is dropped;
//   - Aggravation loses its time damage;
//   - detail 25 (an attack with extra HP/MP/SP damage) becomes detail 21, the
//     standard attack; AddDamage is read from its own fields, so the effects
//     are identical;
//   - descriptions follow the real spell values: skill_text_fixes.csv holds each
//     rewrite as id, level, current text, new text.
const (
	skillRecordSize   = 116
	skillSlotsPerID   = 9
	skillPrevOffset   = 52 // uint16 previous skill, the chain relation
	skillTimeDamageHP = 75 // int32
)

// retainedLevels look like placeholders but consumables use them (an item's
// Range and AttackTime name a skill and level): the Strength to Wisdom Remedy
// level 3 items, the Mini Sun EXP Stone level 2 and the Little Elkid Clone
// items. Their descriptions are corrected instead.
var retainedLevels = map[SkillKey]bool{{ID: 250, Level: 2}: true, {ID: 274, Level: 3}: true, {ID: 275, Level: 3}: true,
	{ID: 280, Level: 3}: true, {ID: 281, Level: 3}: true, {ID: 282, Level: 3}: true, {ID: 283, Level: 3}: true, {ID: 284, Level: 3}: true, {ID: 285, Level: 3}: true}

// previousSkills is the original PrevSkillID of the skills that have one: that
// skill must be cooling down for the skill to be usable.
var previousSkills = map[int]uint16{61: 104, 64: 150, 79: 135, 94: 79, 104: 134, 135: 134, 145: 134, 150: 145,
	161: 61, 302: 39, 309: 63, 310: 63, 311: 150, 312: 61, 313: 79}

var levelSuffix = regexp.MustCompile(`(?i)\s*Lv\.?\s*\d+$`)

// requireCooldown says which skill has to be cooling down (once).
func requireCooldown(desc, prev string) string {
	const marker = " is cooling down."
	if prev == "" || strings.Contains(desc, marker) {
		return desc
	}
	desc = strings.TrimSpace(desc)
	if !strings.HasSuffix(desc, ".") {
		desc += "."
	}
	return desc + " Can only be used while " + prev + marker
}

//go:embed skill_text_fixes.csv
var skillTextFixesCSV []byte

var (
	chainPrefix  = regexp.MustCompile(`^Chain skill (?:that requires|used after) [^;]+; `)
	chainSuffix  = regexp.MustCompile(` but the following chain skill must be used in \d+ seconds\.$`)
	enemyOnly    = regexp.MustCompile(` ?Only usable on (?:players from enemy realm|enemy players not monsters)\.`)
	notAvailable = regexp.MustCompile(`(?i)not available|non-existen|non-existan`)
)

// SkillCatalogReport counts what NormalizeSkillCatalog changed.
type SkillCatalogReport struct {
	DeadLevels      []SkillKey // blanked
	ChainTexts      int
	PrevRestored    []SkillKey
	ManaStaminaText int
	TimeDamage      []SkillKey
	TextFixes       int
	DetailMoved     []SkillKey
	FullyDeadSkills []uint16 // left untouched
}

func (r SkillCatalogReport) String() string {
	return fmt.Sprintf("dead levels removed: %d; chain descriptions rewritten: %d; previous skill restored: %d; Mana/Stamina Shot texts: %d; Aggravation time damage cleared: %d; detail 25 -> 21: %d; descriptions corrected: %d; skills with no working level left alone: %v",
		len(r.DeadLevels), r.ChainTexts, len(r.PrevRestored), r.ManaStaminaText, len(r.TimeDamage), len(r.DetailMoved), r.TextFixes, r.FullyDeadSkills)
}

type skillSlot struct {
	id, slot         int
	nameRaw, descRaw []byte // length prefix included
	name, desc       string
	rec              []byte
	changed          bool
}

func (s *skillSlot) blank() bool { return !anyByte(s.rec) }
func (s *skillSlot) key() SkillKey {
	return SkillKey{ID: uint16(s.id), Level: s.rec[0]}
}
func (s *skillSlot) dead() bool {
	return s.rec[0] > 0 && notAvailable.MatchString(s.desc) && !retainedLevels[s.key()]
}
func anyByte(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return true
		}
	}
	return false
}

func lengthPrefixed(s string) []byte {
	return append(binary.LittleEndian.AppendUint32(nil, uint32(len(s)+1)), append([]byte(s), 0)...)
}

// unchain removes chain wording from a description (NUL excluded).
func unchain(d string) (string, bool) {
	out := d
	if strings.HasPrefix(out, "Opening skill of chain skill that deals ") {
		out = "Deals " + strings.TrimPrefix(out, "Opening skill of chain skill that deals ")
	}
	out = chainPrefix.ReplaceAllString(out, "")
	out = chainSuffix.ReplaceAllString(out, ".")
	return out, out != d
}

func NormalizeSkillCatalog(raw []byte) ([]byte, SkillCatalogReport, error) {
	return normalizeSkillCatalog(raw, skillTextFixesCSV)
}

func normalizeSkillCatalog(raw, textFixes []byte) ([]byte, SkillCatalogReport, error) {
	var rep SkillCatalogReport
	plain, err := Decrypt(raw)
	if err != nil {
		return nil, rep, err
	}
	c := &cursor{b: plain}
	groups := c.count(skillSlotsPerID * 124)
	slots := make([]*skillSlot, 0, groups*skillSlotsPerID)
	for id := 1; id <= groups && c.err == nil; id++ {
		for slot := 1; slot <= skillSlotsPerID && c.err == nil; slot++ {
			start := c.off
			name := c.str()
			nameEnd := c.off
			desc := c.str()
			descEnd := c.off
			rec := append([]byte(nil), c.take(skillRecordSize)...)
			if c.err != nil {
				break
			}
			slots = append(slots, &skillSlot{id: id, slot: slot, nameRaw: plain[start:nameEnd], descRaw: plain[nameEnd:descEnd],
				name: strings.TrimRight(name, "\x00"), desc: strings.TrimRight(desc, "\x00"), rec: rec})
		}
	}
	if c.err != nil {
		return nil, rep, c.err
	}
	if c.off != len(plain) {
		return nil, rep, fmt.Errorf("Skill.SData trailing data")
	}

	working := map[int]bool{}
	for _, s := range slots {
		if !s.blank() && !s.dead() {
			working[s.id] = true
		}
	}
	dead := map[int]bool{}
	for _, s := range slots {
		if s.dead() {
			dead[s.id] = true
		}
	}
	for id := range dead {
		if !working[id] {
			rep.FullyDeadSkills = append(rep.FullyDeadSkills, uint16(id))
		}
	}
	sort.Slice(rep.FullyDeadSkills, func(i, j int) bool { return rep.FullyDeadSkills[i] < rep.FullyDeadSkills[j] })

	for _, s := range slots {
		switch {
		case s.dead() && working[s.id]:
			rep.DeadLevels = append(rep.DeadLevels, s.key())
			s.name, s.desc, s.rec, s.changed = "", "", make([]byte, skillRecordSize), true
			continue
		case s.blank():
			continue
		}
		if text, ok := unchain(s.desc); ok {
			if strings.Contains(strings.ToLower(text), "chain") {
				return nil, rep, fmt.Errorf("skill %d level %d: chain wording left in %q", s.id, s.rec[0], text)
			}
			s.desc, s.changed = text, true
			rep.ChainTexts++
		}
		if prev, ok := previousSkills[s.id]; ok {
			if binary.LittleEndian.Uint16(s.rec[skillPrevOffset:]) != prev {
				binary.LittleEndian.PutUint16(s.rec[skillPrevOffset:], prev)
				rep.PrevRestored = append(rep.PrevRestored, s.key())
				s.changed = true
			}
		}
		if s.id == 115 || s.id == 314 {
			if text := enemyOnly.ReplaceAllString(s.desc, ""); text != s.desc {
				s.desc, s.changed = text, true
				rep.ManaStaminaText++
			}
		}
		if s.id == 150 && int32(binary.LittleEndian.Uint32(s.rec[skillTimeDamageHP:])) != 0 {
			binary.LittleEndian.PutUint32(s.rec[skillTimeDamageHP:], 0)
			rep.TimeDamage = append(rep.TimeDamage, s.key())
			s.changed = true
		}
		if s.rec[21] == 1 && binary.LittleEndian.Uint16(s.rec[22:]) == 25 {
			binary.LittleEndian.PutUint16(s.rec[22:], 21)
			rep.DetailMoved = append(rep.DetailMoved, s.key())
			s.changed = true
		}
	}

	if err := applyTextFixes(slots, textFixes, &rep); err != nil {
		return nil, rep, err
	}
	// After the rewrites, which match the text without this sentence.
	skillNames := map[int]string{}
	for _, s := range slots {
		if _, ok := skillNames[s.id]; !ok && !s.blank() {
			skillNames[s.id] = levelSuffix.ReplaceAllString(strings.Join(strings.Fields(s.name), " "), "")
		}
	}
	for _, s := range slots {
		if prev, ok := previousSkills[s.id]; ok && !s.blank() {
			if text := requireCooldown(s.desc, skillNames[int(prev)]); text != s.desc {
				s.desc, s.changed = text, true
			}
		}
	}

	out := binary.LittleEndian.AppendUint32(nil, uint32(groups))
	for _, s := range slots {
		if s.changed {
			s.nameRaw, s.descRaw = lengthPrefixed(s.name), lengthPrefixed(s.desc)
		}
		out = append(append(append(out, s.nameRaw...), s.descRaw...), s.rec...)
	}
	if _, err := ParseSkills(out); err != nil {
		return nil, rep, err
	}
	return out, rep, nil
}

// applyTextFixes rewrites the descriptions listed in skill_text_fixes.csv. A
// row applies when the slot still has its old text and is skipped when it has
// the new one; anything else means the catalog is not the expected one.
func applyTextFixes(slots []*skillSlot, fixes []byte, rep *SkillCatalogReport) error {
	rows, err := csv.NewReader(bytes.NewReader(fixes)).ReadAll()
	if err != nil {
		return err
	}
	byKey := map[SkillKey]*skillSlot{}
	for _, s := range slots {
		if !s.blank() {
			byKey[s.key()] = s
		}
	}
	for _, row := range rows {
		if len(row) != 4 {
			return fmt.Errorf("skill_text_fixes.csv: malformed row %q", row)
		}
		id, err1 := strconv.Atoi(row[0])
		level, err2 := strconv.Atoi(row[1])
		if err1 != nil || err2 != nil {
			return fmt.Errorf("skill_text_fixes.csv: bad skill %q/%q", row[0], row[1])
		}
		s := byKey[SkillKey{ID: uint16(id), Level: uint8(level)}]
		switch {
		case s == nil:
			return fmt.Errorf("skill_text_fixes.csv: missing skill %d level %d", id, level)
		case s.desc == row[3], strings.HasPrefix(s.desc, strings.TrimSuffix(row[3], ".")) && strings.Contains(s.desc, " is cooling down."):
		case s.desc == row[2]:
			s.desc, s.changed = row[3], true
			rep.TextFixes++
		default:
			return fmt.Errorf("skill %d level %d: description is %q, expected %q", id, level, s.desc, row[2])
		}
	}
	return nil
}
