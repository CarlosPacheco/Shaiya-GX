package world

import (
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/skillrange"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

const (
	detailFireThorn = 84              // buff: thorns around the caster
	fireThornTick   = 3 * time.Second // the pulse interval
)

func isFireThorn(b *ActiveBuff) bool {
	return b.Effect == typeEffectBuff && b.Detail == detailFireThorn
}

// tickFireThorn runs Fire Thorn (TypeEffect 4, TypeDetail 84). While the buff
// lasts, every three seconds each hostile mob and user within ApplyRange of
// the caster takes the skill's DamageHP as fixed damage: no hit roll, no stat
// formula, no defense, no absorption. Mobs also gain aggro on the caster.
func (s *Session) tickFireThorn(now time.Time) {
	if s.char == nil || s.dead {
		return
	}
	var pulses []*db.SkillDef
	for i := range s.buffs {
		b := &s.buffs[i]
		if !isFireThorn(b) || b.nextTickAt.IsZero() || now.Before(b.nextTickAt) {
			continue
		}
		b.nextTickAt = b.nextTickAt.Add(fireThornTick)
		if b.nextTickAt.Before(now) {
			b.nextTickAt = now.Add(fireThornTick)
		}
		if def := db.MustDefs().Skill(b.SkillID, b.SkillLevel); def != nil && def.DamageHP > 0 {
			pulses = append(pulses, def)
		}
	}
	for _, def := range pulses {
		s.fireThornPulse(def)
	}
}

func (s *Session) fireThornPulse(def *db.SkillDef) {
	ch := s.char
	radius := float32(def.ApplyRange)
	damage := uint32(def.DamageHP)
	stats := s.stats()
	center := skillrange.Position{ch.PosX, ch.PosY, ch.PosZ}
	for _, id := range s.skillMobsWithin(ch.PosX, ch.PosY, ch.PosZ, radius) {
		mob, ok := MobByID(ch.MapID, id)
		md := db.MustDefs().Mobs[mob.MobID]
		if !ok || md == nil || mobFriendly(md, s.faction) {
			continue
		}
		SetMobAggro(ch.MapID, id, ch.ID)
		r, ok := DamageMobFrom(ch.MapID, id, damage, ch.ID)
		if !ok {
			continue
		}
		s.lastCombatAt = time.Now()
		AddMobAggro(ch.MapID, id, ch.ID, hitAggro(&stats, def.Aggro))
		sendNear(ch.MapID, r.Mob.PosX, r.Mob.PosZ, CombatRadius,
			buildMobPeriodicDamage(r.Mob.ID, def.SkillID, def.SkillLevel, uint16(min(r.DamageHP, 65535)), 0, 0), "0x0607 fire thorn damage")
		s.resolveMobDamage(&r)
	}
	for _, p := range s.hostileUsersWithin(ch.PosX, ch.PosZ, radius) {
		c := p.Snapshot()
		if s.skillAreaContains(center, skillrange.Position{c.PosX, c.PosY, c.PosZ}, radius) {
			s.lastCombatAt = time.Now()
			p.QueueUserHit(pendingUserHit{AttackerID: ch.ID, Fixed: true, Damage: [3]uint32{damage}, Skill: def})
		}
	}
}

// applyFixedHit is the target side of a fixed-damage skill hit (Fire Thorn):
// the damage is final, so no accuracy, defense or absorption applies. Nearby
// users get the periodic damage packet 0x050F; true when this character died.
func (s *Session) applyFixedHit(h pendingUserHit) bool {
	ch := s.char
	killed := s.takeUserSkillDamage(h.AttackerID, h.Damage)
	w := packet.New(packet.SkillKeep)
	w.U32(ch.ID)
	w.U16(h.Skill.SkillID)
	w.U8(h.Skill.SkillLevel)
	for _, v := range h.Damage {
		w.U16(uint16(min(v, 65535)))
	}
	sendNear(ch.MapID, ch.PosX, ch.PosZ, CombatRadius, w.MustFinish(), "0x050F fire thorn damage")
	s.afterUserDamage(h.AttackerID, killed)
	return killed
}
