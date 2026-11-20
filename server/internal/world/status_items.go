package world

import (
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

const (
	protectFaint uint16 = 1 << 7
	protectDeath uint16 = 1 << 8
)

// EP4.5 CUser::ItemUseRemState (0x4653e0) selects the original
// StateType, with Perfect Cure matching every debuff.
func statusItemRule(special uint8) (states, protection uint16, allDebuffs bool) {
	switch {
	case special == 7:
		return protectFaint, protectFaint, false
	case special == 8:
		return protectDeath, protectDeath, false
	case special >= 1 && special <= 15:
		return 1 << special, 0, false
	}
	switch special {
	case 18:
		return 1<<9 | 1<<10 | 1<<11, 0, false
	case 19:
		return 1<<1 | 1<<2 | 1<<5 | 1<<6, 0, false
	case 20:
		return 1<<3 | 1<<4, 0, false
	case 21:
		return 1<<14 | 1<<15, 0, false
	case 22:
		return 1<<12 | 1<<13, 0, false
	case 23:
		return protectFaint | protectDeath, protectFaint | protectDeath, false
	case 26:
		return 0, protectFaint | protectDeath, true
	default:
		return 0, 0, false
	}
}

func (s *Session) useStatusItem(index int, def *db.ItemDef) bool {
	states, protection, allDebuffs := statusItemRule(def.Special)
	if s.dead || s.char.CurrentHP == 0 {
		return false
	}
	now := time.Now()
	if delay := cooldownForReqIg(def.ReqIg); delay > 0 && now.Sub(s.itemCooldowns[def.ReqIg]) < delay {
		return false
	}
	match := func(buff *ActiveBuff) bool {
		if allDebuffs {
			return buff.Effect == typeEffectDebuff
		}
		skill := db.MustDefs().EffectSkill(buff.SkillID, buff.SkillLevel)
		return skill != nil && states&(1<<skill.StateType) != 0
	}
	rows := s.buffSnapshotExcept(match)
	var item db.Item
	if !s.inventoryOperationWithCommit(func() {
		var err error
		item, err = s.consumeOne(index)
		s.operation.err = err
	}, func(change db.CharacterChange) error {
		return db.CommitStatusItem(change, rows)
	}) {
		return false
	}
	s.removeEffects(match, "status item")
	s.statusItemProtection |= protection
	if cooldownForReqIg(def.ReqIg) > 0 {
		if s.itemCooldowns == nil {
			s.itemCooldowns = make(map[uint8]time.Time)
		}
		s.itemCooldowns[def.ReqIg] = now
	}
	s.send(buildUseItemResult(s.char.ID, &item), "0x050A status item used")
	return true
}

// The original player and mob hit checks consume the matching flag before
// rolling success (0x456b39 and 0x4a2419).
func (s *Session) blocksStatusSkill(def *db.SkillDef) bool {
	mask := uint16(1) << def.StateType
	blocked := s.statusItemProtection&mask != 0
	s.statusItemProtection &^= mask
	return blocked
}
