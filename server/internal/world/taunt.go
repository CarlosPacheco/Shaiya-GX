package world

import (
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

const detailTaunt = 69 // buff: pure aggro on the mobs around the caster

// hitAggro is the aggro one hit adds to a mob's ledger (CMob::Damage): the
// attacker's REC times ten plus the skill's Arrow, whatever the damage dealt.
// Normal attacks have no skill, so only the REC term.
func hitAggro(stats *CharacterStats, arrow uint16) int64 {
	return int64(stats.Rec)*10 + int64(arrow)
}

// tauntMobs is CUser::TauntMob (AddApplySkillBuff, TypeDetail 69): every
// living hostile mob within ApplyRange of the caster turns to the caster at
// once and its ledger gains the Arrow plus ten times the caster's REC, with no
// damage and no other effect. The mob's next retarget may still prefer whoever
// holds more aggro.
func (s *Session) tauntMobs(def *db.SkillDef, stats *CharacterStats) {
	ch := s.char
	amount := hitAggro(stats, def.Aggro)
	taunted := false
	for _, id := range s.mobsNear(ch.PosX, ch.PosZ, float32(def.ApplyRange)) {
		mob, ok := MobByID(ch.MapID, id)
		md := db.MustDefs().Mobs[mob.MobID]
		if !ok || mob.CurrentHP == 0 || md == nil || mobFriendly(md, s.faction) {
			continue
		}
		ForceMobTarget(ch.MapID, id, ch.ID)
		if amount > 0 {
			AddMobAggro(ch.MapID, id, ch.ID, amount)
		}
		taunted = true
	}
	if taunted {
		s.lastCombatAt = time.Now()
	}
}
