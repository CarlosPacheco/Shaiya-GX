package world

import (
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

const (
	detailPeriodicHeal = 81
	periodicHealTick   = 3 * time.Second
)

// CUser::ChkKeepSkill applies detail 81 to the buff holder. The catalog's
// TimeHeal values are per three seconds and do not scale with caster stats.
func (s *Session) tickPeriodicHeal(b *ActiveBuff, now time.Time) {
	if b.nextTickAt.IsZero() {
		return
	}
	start := b.nextTickAt.Add(-periodicHealTick)
	elapsed := now.Sub(start).Truncate(periodicHealTick)
	if !b.MapCurse && !now.Before(b.EndsAt) {
		elapsed = b.EndsAt.Sub(start)
	}
	if elapsed <= 0 {
		return
	}
	// Consume every due interval once, including any final partial interval.
	b.nextTickAt = b.nextTickAt.Add(elapsed)
	hp, mp, sp := s.healDelta(periodicHealAmount(b.HealHP, elapsed), periodicHealAmount(b.HealMP, elapsed), periodicHealAmount(b.HealSP, elapsed))
	if hp|mp|sp == 0 {
		return
	}
	s.persist()
	if s.persistenceFailed {
		return
	}
	w := packet.New(packet.SkillKeep)
	w.U32(s.char.ID)
	w.U16(b.SkillID)
	w.U8(b.SkillLevel)
	w.U16(uint16(min(hp, 65535)))
	w.U16(uint16(min(mp, 65535)))
	w.U16(uint16(min(sp, 65535)))
	frame := w.MustFinish()
	const label = "0x050F buff heal tick"
	s.send(frame, label)
	for _, p := range PlayersNearExcluding(s.char.MapID, s.char.ID, s.char.PosX, s.char.PosZ, CombatRadius) {
		p.Send(frame, label)
	}
	st := s.stats()
	s.send(buildCurrentHitpoints(&st), "0x0521 current hp/mp/sp after buff heal tick")
}

func periodicHealAmount(perTick uint32, elapsed time.Duration) uint32 {
	whole := satMul(perTick, uint32(elapsed/periodicHealTick))
	partial := uint32(uint64(perTick) * uint64(elapsed%periodicHealTick) / uint64(periodicHealTick))
	return satAdd(whole, partial)
}
