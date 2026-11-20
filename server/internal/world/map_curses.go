package world

import (
	"slices"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func mapCurseDef(defs *db.Defs, id uint16) *db.SkillDef {
	d := defs.Skill(id, 1)
	if d != nil && (d.TypeEffect == typeEffectBuff || d.TypeEffect == typeEffectDebuff) {
		return d
	}
	return nil
}

func (s *Session) refreshMapCurses(publish bool) {
	ids := mapConfig.Zones[realMap(s.char.MapID)].MapCurses
	if publish && slices.ContainsFunc(s.buffs, func(b ActiveBuff) bool { return b.MapCurse }) {
		s.removeEffectsInternal(func(b *ActiveBuff) bool { return b.MapCurse }, "leaving map", true)
	}
	if ids == [3]uint16{} {
		return
	}
	// The snapshots on character selection carry the effects without extra
	// stock BuffAdd packets. At a map change publish them and refresh stats.
	before := s.speeds()
	added := false
	var seen [3]uint16
	for i, id := range ids {
		if id == 0 {
			continue
		}
		duplicate := false
		for _, old := range seen[:i] {
			if old == id {
				duplicate = true
			}
		}
		if duplicate {
			continue
		}
		seen[i] = id
		def := mapCurseDef(db.MustDefs(), id)
		if def == nil {
			continue
		}
		if publish {
			s.removeEffects(func(b *ActiveBuff) bool { return b.SkillID == id }, "replaced by map curse")
		} else {
			s.buffs = slices.DeleteFunc(s.buffs, func(b ActiveBuff) bool { return b.SkillID == id })
		}
		var buffID uint32
		for _, b := range s.buffs {
			buffID = max(buffID, b.ID)
		}
		b := newBuff(buffID+1, def, time.Now(), time.Time{}, s.log)
		b.MapCurse = true
		s.buffs = append(s.buffs, b)
		added = true
		if publish {
			s.send(buildBuffAdd(b.ID, b.SkillID, b.SkillLevel), "0x050D map curse")
		}
	}
	if added && publish {
		s.syncVisibility()
		s.sendBuffStats("map curses")
		if after := s.speeds(); after != before {
			s.broadcastSpeed(after)
		}
	}
}
