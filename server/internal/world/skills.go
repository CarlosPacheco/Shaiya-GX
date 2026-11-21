package world

import (
	"slices"
	"sort"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/skilltiming"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

const (
	skillBarCount     = 5
	skillBarSlotCount = 10
	quickSlotSkillBag = 100
	maxQuickSlots     = 128

	mountSlot      = 13
	mountItemType  = 42
	unmountedShape = 0x00
)

func buildCharacterSkills(skills []db.CharacterSkill) []byte {
	w := packet.New(packet.CharacterSkills)
	skills = skills[:min(len(skills), 255)]
	w.U8(uint8(len(skills)))
	for _, s := range skills {
		w.U16(s.SkillID)
		w.U8(s.Level)
		w.U8(s.Number)
		w.I32(int32(s.Delay))
	}
	return w.MustFinish()
}

func isValidQuickSlot(q db.QuickSlot) bool {
	return q.QuickBar < skillBarCount && q.QuickSlot < skillBarSlotCount &&
		((q.Bag <= InventoryLastBag && q.Number < 24) || (q.Bag == quickSlotSkillBag && q.Number > 0))
}

func buildCharacterSkillBar(slots []db.QuickSlot) []byte {
	var valid []db.QuickSlot
	for _, q := range slots {
		if isValidQuickSlot(q) && len(valid) < maxQuickSlots {
			valid = append(valid, q)
		}
	}
	w := packet.New(packet.CharacterSkillBar)
	w.U8(uint8(len(valid)))
	for _, q := range valid {
		w.U8(q.QuickBar)
		w.U8(q.QuickSlot)
		w.U8(q.Bag)
		w.U16(q.Number)
	}
	return w.MustFinish()
}

// parseSkillBar (0x010B): count, then bar, slot, bag, number:u16.
func parseSkillBar(p []byte) []db.QuickSlot {
	if len(p) < 3 {
		return nil
	}
	var out []db.QuickSlot
	off := 3
	for range min(int(p[2]), maxQuickSlots) {
		if len(p) < off+5 {
			break
		}
		n, _ := packet.U16(p[off+3:])
		q := db.QuickSlot{QuickBar: p[off], QuickSlot: p[off+1], Bag: p[off+2], Number: n}
		if isValidQuickSlot(q) {
			out = append(out, q)
		}
		off += 5
	}
	return out
}

func (s *Session) handleSkillBar(p []byte) {
	if s.char == nil {
		return
	}
	if len(p) < 3 || int(p[2]) > maxQuickSlots || len(p) != 3+int(p[2])*5 {
		return
	}
	slots := parseSkillBar(p)
	if slices.Equal(slots, s.quickSlots) {
		return
	}
	if err := db.SaveQuickSlots(s.char.ID, slots); err != nil {
		s.log.Printf("GX Server World failed to save quick slots: %v", err)
		return
	}
	s.quickSlots = slots

}

// buildLearnSkillError (0x0209): reason, skill id, level.
func buildLearnSkillError(reason uint8, skillID uint16, level uint8) []byte {
	w := packet.New(packet.LearnNewSkill)
	w.U8(reason)
	w.U8(0)
	w.U16(skillID)
	w.U8(level)
	return w.MustFinish()
}

// skillCountryAllows is EnableLearnSkill's Country byte by family (0 human,
// 1 elf, 2 vail, 3 deatheater).
func skillCountryAllows(country, family uint8) bool {
	switch country {
	case 0:
		return family == 0
	case 1:
		return family == 1
	case 2:
		return family == 0 || family == 1
	case 3:
		return family == 3
	case 4:
		return family == 2
	case 5:
		return family == 2 || family == 3
	}
	return true
}

func buildLearnSkillResult(ok bool, number uint8, skillID uint16, level uint8) []byte {
	w := packet.New(packet.LearnNewSkill)
	w.U8(resultByte(ok))
	w.U8(number)
	w.U16(skillID)
	w.U8(level)
	return w.MustFinish()
}

// handleLearnSkill (0x0209): skill id:u16, level.
func (s *Session) handleLearnSkill(p []byte) {
	if len(p) < 5 || s.char == nil {
		return
	}
	ch := s.char
	skillID, _ := packet.U16(p[2:])
	level := p[4]
	// CUser::LearnSkill / EnableLearnSkill (0x004842e0 / 0x00484080):
	// levels 1-3; a new skill starts at level 1 and a known one goes to
	// any higher level; then skill points (error 1), level (2), and the
	// country, job and TypeShow (3). No prerequisite skill.
	reject := func(code uint8, why string) {
		s.log.Printf("GX Server World rejected learn skill skill_id=%d level=%d: %s", skillID, level, why)
		s.send(buildLearnSkillError(code, skillID, level), "0x0209 learn skill rejected")
	}
	def := db.MustDefs().Skill(skillID, level)
	if level < 1 || level > 3 || def == nil {
		reject(3, "no such skill/level")
		return
	}
	existing := -1
	for i, k := range s.skills {
		if k.SkillID == skillID {
			existing = i
		}
	}
	if existing >= 0 {
		if s.skills[existing].Level >= level {
			reject(3, "level already known")
			return
		}
	} else if level != 1 {
		return
	}
	cost := uint16(def.SkillPointCost)
	switch {
	case ch.SkillPoint < cost:
		reject(1, "not enough skill points")
		return
	case ch.Level < def.ReqLevel:
		reject(2, "level too low")
		return
	case !skillCountryAllows(def.Country, ch.Race):
		s.log.Printf("GX Server World rejected learn skill skill_id=%d: country %d", skillID, def.Country)
		return
	case int(ch.Class) >= len(def.ClassEligible) || !def.ClassEligible[ch.Class] || def.TypeShow == 0:
		reject(3, "job or hidden skill")
		return
	}
	var learned db.CharacterSkill
	var previous *db.CharacterSkill
	if existing >= 0 {
		copy := s.skills[existing]
		previous = &copy
		learned = copy
		learned.Level, learned.Delay = level, 0
	} else {
		var number uint8
		for _, k := range s.skills {
			number = max(number, k.Number)
		}
		if number == 255 {
			reject(3, "skill slots exhausted")
			return
		}
		learned = db.CharacterSkill{SkillID: skillID, Level: level, Number: number + 1}
	}
	learned.Delay = s.remainingSkillDelay(skillID)
	after := *ch
	after.SkillPoint -= cost
	if err := db.CommitLearnSkill(*ch, after, previous, learned); err != nil {
		s.logErr("learned skill transaction", err)
		reject(3, "database error")
		return
	}
	ch.SkillPoint = after.SkillPoint
	if existing >= 0 {
		s.skills[existing] = learned
	} else {
		s.skills = append(s.skills, learned)
		sort.Slice(s.skills, func(i, j int) bool { return s.skills[i].Number < s.skills[j].Number })
	}
	s.syncPlayer()

	s.log.Printf("GX Server World learned skill number=%d skill_id=%d level=%d remaining_skill_point=%d", learned.Number, skillID, level, ch.SkillPoint)
	s.send(buildLearnSkillResult(true, learned.Number, skillID, level), "0x0209 learn skill ok")
	s.sendPlayerFeedback(proto.PlayerSkillFeedback)
	if learned.Delay > 0 {
		s.sendSkillCooldown(skillID, uint16(learned.Delay), skilltiming.ReuseSeconds(def.ResetTimeSeconds))
	}
	s.syncPlayerSpeed(false)
}

// handleSetStatPoint (0x0208): six u16 deltas str, dex, rec, int, wis, luc.
func (s *Session) handleSetStatPoint(p []byte) {
	if len(p) < 14 || s.char == nil {
		return
	}
	ch := s.char
	var d [6]uint16
	total := uint32(0)
	for i := range d {
		d[i], _ = packet.U16(p[2+2*i:])
		total += uint32(d[i])
	}
	if total == 0 || total > uint32(ch.StatPoint) {
		s.log.Printf("GX Server World rejected stat points total=%d available=%d", total, ch.StatPoint)
		return
	}
	// A stat set to the DB cap by a GM must remain persistible after a
	// normal allocation too. Refuse the whole request before spending points.
	for i, base := range []uint16{ch.Str, ch.Dex, ch.Rec, ch.Int, ch.Wis, ch.Luc} {
		if d[i] > 0 && uint32(base)+uint32(d[i]) > db.BaseStatCap {
			s.log.Printf("GX Server World rejected stat allocation beyond base-stat limit")
			return
		}
	}
	add := func(v *uint16, n uint16) { *v += n }
	add(&ch.Str, d[0])
	add(&ch.Dex, d[1])
	add(&ch.Rec, d[2])
	add(&ch.Int, d[3])
	add(&ch.Wis, d[4])
	add(&ch.Luc, d[5])
	ch.StatPoint -= uint16(total)
	s.persist()
	stats := s.stats()
	w := packet.New(packet.SetStatPoint)
	for _, v := range d {
		w.U16(v)
	}
	s.send(w.MustFinish(), "0x0208 stat point applied")
	s.send(buildAdditionalStats(&stats), "0x0526 additional stats after stat allocation")
	if d[2] > 0 {
		s.send(buildMaxEnergy(ch, 0, stats.MaxHP), "0x050B max hp")
	}
	if d[1] > 0 {
		s.send(buildMaxEnergy(ch, 1, stats.MaxSP), "0x050B max sp")
	}
	if d[4] > 0 {
		s.send(buildMaxEnergy(ch, 2, stats.MaxMP), "0x050B max mp")
	}
	s.log.Printf("GX Server World applied stat points character_id=%d deltas=%v remaining=%d", ch.ID, d, ch.StatPoint)
}
