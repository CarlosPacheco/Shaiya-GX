package world

import (
	"math"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

const (
	detailFireThorn = 84              // buff: thorns around the caster
	fireThornTick   = 2 * time.Second // the pulse interval
	// pulseHeightReach is how far above or below the pulse's centre a target may
	// stand and still burn: the height counts, but leniently (slopes and steps
	// must not narrow the ground reach).
	pulseHeightReach = 4
	// fireThornRange is how far the thorns reach, whatever the skill's ApplyRange.
	fireThornRange = 6
)

func isFireThorn(b *ActiveBuff) bool {
	return b.Effect == typeEffectBuff && b.Detail == detailFireThorn
}

// tickFireThorn runs Fire Thorn (TypeEffect 4, TypeDetail 84). While the buff
// lasts, every two seconds each hostile mob and user within ApplyRange of
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
	s.fixedDamagePulse(def, ch.PosX, ch.PosY, ch.PosZ, fireThornRange)
}

// fixedDamagePulse hurts every hostile mob and user within ApplyRange of a
// point with the skill's DamageHP as fixed damage (Fire Thorn around the
// caster, Fire Bush around its curtain of fire). The client plays DA_self_fire
// on each target.
func (s *Session) fixedDamagePulse(def *db.SkillDef, x, y, z, reach float32) {
	ch := s.char
	radius := float32(def.ApplyRange)
	if reach > 0 { // a reach of its own, wider than the skill's ApplyRange
		radius = reach
	}
	damage := uint32(def.DamageHP)
	stats := s.stats()
	for _, id := range s.mobsNear(x, z, radius, y, pulseHeightReach) {
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
	for _, p := range s.hostileUsersWithin(x, z, radius) {
		if c := p.Snapshot(); math.Abs(float64(c.PosY-y)) > pulseHeightReach {
			continue
		}
		s.lastCombatAt = time.Now()
		p.QueueUserHit(pendingUserHit{AttackerID: ch.ID, Fixed: true, Damage: [3]uint32{damage}, Skill: def})
	}
}

// applyFixedHit is the target side of a fixed-damage skill hit (Fire Thorn):
// the damage is final, so no accuracy, defense or absorption applies. Nearby
// users get the periodic damage packet 0x050F; true when this character died.
func (s *Session) applyFixedHit(h pendingUserHit) bool {
	ch := s.char
	killed := s.takeUserSkillDamage(h.AttackerID, h.Damage)
	shown := h.Damage
	if shown[0] > 0 { // Health Assistant moved part of it to SP / MP
		shown = [3]uint32{s.assisted[0], shown[1] + s.assisted[1], shown[2] + s.assisted[2]}
	}
	w := packet.New(packet.SkillKeep)
	w.U32(ch.ID)
	w.U16(h.Skill.SkillID)
	w.U8(h.Skill.SkillLevel)
	for _, v := range shown {
		w.U16(uint16(min(v, 65535)))
	}
	sendNear(ch.MapID, ch.PosX, ch.PosZ, CombatRadius, w.MustFinish(), "0x050F fire thorn damage")
	s.afterUserDamage(h.AttackerID, killed)
	return killed
}
