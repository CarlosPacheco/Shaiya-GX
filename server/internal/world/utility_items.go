package world

import (
	"slices"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

const (
	specialSkillReset         = 33
	specialTargetResurrection = 41
	// EP4.5 checks the same zone without an explicit distance in ItemUse/LifeSkill.
	// GX caps use at the combat visibility distance.
	targetResurrectionRange = CombatRadius
)

func (s *Session) resetSkillsWithItem(index int) bool {
	ch := s.char
	if s.dead || ch.CurrentHP == 0 {
		return false
	}
	perLevel := 3
	switch ch.Mode {
	case growHard:
		perLevel = 4
	case growUltimate:
		perLevel = 5
	}
	var item db.Item
	if !s.inventoryOperationWithCommit(func() {
		s.char.SkillPoint = uint16(min(65535, 5+max(0, int(ch.Level)-1)*perLevel))
		var err error
		item, err = s.consumeOne(index)
		s.operation.err = err
	}, func(change db.CharacterChange) error {
		return db.CommitSkillReset(change, s.skillCooldowns)
	}) {
		return false
	}
	s.skills = nil
	s.quickSlots = slices.DeleteFunc(s.quickSlots, func(q db.QuickSlot) bool { return q.Bag == 100 })
	s.cast, s.pendingCombo, s.dash = nil, nil, nil
	s.lastSkillID = 0
	s.syncPlayer()
	s.sendBuffStats("skill reset")
	s.syncPlayerSpeed(false)
	stats := s.stats()
	s.send(buildCharacterDetails(ch, &stats), "0x0105 skill reset details")
	s.send(buildCharacterSkills(s.skills), "0x0108 skill reset")
	s.send(buildCharacterSkillBar(s.quickSlots), "0x010B skill reset shortcuts")
	s.send(buildUseItemResult(ch.ID, &item), "0x050A skill reset stone")
	return true
}

func (s *Session) resurrectWithItem(index int, id uint32) bool {
	ch := s.char
	if s.dead || ch.CurrentHP == 0 || id == ch.ID {
		return false
	}
	player := FindPlayerByID(id)
	if player == nil || player.owner == nil || player.owner == s {
		return false
	}
	target := player.owner
	if !target.stateMu.TryLock() {
		return false
	}
	defer target.stateMu.Unlock()
	dead := target.char
	if target.player != player || dead == nil || !target.dead || target.faction != s.faction || dead.MapID != ch.MapID {
		return false
	}
	dx, dy, dz := dead.PosX-ch.PosX, dead.PosY-ch.PosY, dead.PosZ-ch.PosZ
	if dx*dx+dy*dy+dz*dz > targetResurrectionRange*targetResurrectionRange {
		return false
	}
	after := *dead
	full := target.stats()
	after.CurrentHP, after.CurrentMP, after.CurrentSP = full.MaxHP, full.MaxMP, full.MaxSP
	changeTarget := db.CharacterChange{Before: *dead, After: after, ItemsBefore: target.items, ItemsAfter: target.items}
	var item db.Item
	if !s.inventoryOperationWithCommit(func() {
		var err error
		item, err = s.consumeOne(index)
		s.operation.err = err
	}, func(change db.CharacterChange) error {
		return db.CommitGameplay([]db.CharacterChange{change, changeTarget}, nil)
	}) {
		return false
	}
	*dead = after
	target.broadcastNear(buildCharacterReviveInit(id), "0x0406 rune corpse leave", true)
	target.dead, target.rebirthAt, target.rebirthParty = false, time.Time{}, false
	player.SetAlive(true)
	player.UpdateCharacter(after)
	target.broadcastNear(buildCharacterResurrect(id, rebirthKindBind, after.Exp, after.PosX, after.PosY, after.PosZ), "0x0551 target resurrection rune", true)
	target.send(buildCharacterReviveHitpoints(id, after.CurrentHP, after.CurrentMP, after.CurrentSP), "0x0505 rune pools")
	target.partyPools()
	s.send(buildUseItemResult(ch.ID, &item), "0x050A target resurrection rune")
	return true
}

func (s *Session) remainingSkillDelay(skillID uint16) int16 {
	left := time.Until(s.skillCooldowns[skillID])
	if left <= 0 {
		return 0
	}
	return int16(min(32767, (left-1)/time.Second+1))
}
