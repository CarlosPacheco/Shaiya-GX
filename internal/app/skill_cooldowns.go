package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/skilltiming"
)

// Confirmed reuse timers drive presentation and suppress duplicate requests.
// Server validation remains authoritative.
func (a *App) confirmedSkills(op proto.Opcode) {
	v := &a.skills
	if op == proto.CharacterSkills {
		a.loadSkills()
		v.queuedSkill = nil
		v.cooldowns = map[uint16]float64{}
		v.cooldownTotal = nil
		v.assign = nil
		v.editing = false
		for _, k := range a.state.Skills {
			delay := max(k.Delay, 0)
			if d, ok := a.skillDefs[sdata.SkillKey{ID: k.ID, Level: k.Level}]; ok && skilltiming.ReuseSeconds(d.ResetTime) == 0 {
				delay = 0
			}
			v.cooldowns[k.ID] = a.clock + float64(delay)
		}
		return
	}
	if op == proto.GXSkillCooldown {
		// The server started or changed a timer: a Potential triggering, a
		// cooldown reduction, or a spell cast while one is active.
		c := a.state.LastSkillCooldown
		if v.cooldowns == nil {
			v.cooldowns = map[uint16]float64{}
		}
		v.cooldowns[c.Skill] = a.clock + float64(c.Seconds)
		if c.Seconds == 0 {
			delete(v.cooldownTotal, c.Skill)
		} else if c.Total > 0 {
			if v.cooldownTotal == nil {
				v.cooldownTotal = map[uint16]float64{}
			}
			v.cooldownTotal[c.Skill] = float64(c.Total)
		}
		return
	}
	if op != proto.SkillUse && op != proto.SkillUseNoTarget && op != proto.SkillUserDrain && op != proto.SkillMobDrain {
		return
	}
	r := a.state.LastSkill
	// 0..3 are hit/critical/miss, 4 announces an area/multi-hit cast.
	// A failed targetless once-skill can also return 3, without starting reuse.
	if r.Source != a.state.SelfID || r.Result > 4 || r.Result == 3 && r.Target == 0 {
		return
	}
	if v.cooldowns[r.ID] > a.clock {
		return
	} // Multiple targets/hits share one reuse period.
	a.loadSkills()
	if v.cooldowns == nil {
		v.cooldowns = map[uint16]float64{}
	}
	d := a.skillDefs[sdata.SkillKey{ID: r.ID, Level: r.Level}]
	v.cooldowns[r.ID] = a.clock + float64(skilltiming.ReuseSeconds(d.ResetTime))
	delete(v.cooldownTotal, r.ID) // an own start is a full ordinary timer
}

// skillCooldownTotal is what a full timer bar means for this skill now: the
// server's figure when it shortened or started the timer, else its reuse.
func (a *App) skillCooldownTotal(id uint16, d sdata.Skill) float64 {
	if total := a.skills.cooldownTotal[id]; total > 0 {
		return total
	}
	return float64(skilltiming.ReuseSeconds(d.ResetTime))
}
