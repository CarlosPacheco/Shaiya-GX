package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"math/rand/v2"
)

func areaShowerSkill(id uint16) bool {
	return id == 53 || id == 68 || id == 71 || id == 162 || id == 163
}

type skillAreaParticle struct {
	pos, velocity render.V3
	instance      *scene.Effect
	grounded      bool
	groundAge     float32
}
type skillAreaShower struct {
	center, side          render.V3
	radius                int
	remaining, credit     float32
	rate, height, lateral float32
	file                  string
	particles             []skillAreaParticle
}

// Game.exe 0x450780/0x430530 creates a fixed world shower. 0x430a80
// emits five falling sequence-zero instances per second for five seconds,
// at signed integer offsets within ApplyRange, then sequence one on terrain.
func (a *App) startSkillAreaShower(r proto.SkillResult, d sdata.Skill, file string) {
	if len(a.wv.areaShowers) >= 16 {
		return
	}
	m, ok := a.impactTransform(skillImpact{target: r.Target, positionOnly: true})
	if !ok {
		return
	}
	center := render.V3{m[12], m[13], m[14]}
	source, ok := a.effectPosition(r.Source)
	if !ok {
		source = center
	}
	dir := center.Sub(source)
	dir[1] = 0
	dir = dir.Norm()
	if dir.Len() == 0 {
		dir = render.V3{0, 0, 1}
	}
	side := render.V3{-dir[2], 0, dir[0]}
	shower := skillAreaShower{center: center, side: side, radius: max(1, int(d.ApplyRange)), remaining: 5, rate: 5, height: 15, lateral: 7.5, file: file}
	if r.ID == 53 {
		shower.remaining, shower.rate, shower.height, shower.lateral = 2, 10, 20, 10
	}
	a.wv.areaShowers = append(a.wv.areaShowers, shower)
	a.playSound(effect.ImpactSound(file), center[0], center[2])
}
func (a *App) updateSkillAreaShowers(dt float32) {
	rows := a.wv.areaShowers[:0]
	for _, shower := range a.wv.areaShowers {
		emitDT := min(dt, max(0, shower.remaining))
		shower.remaining -= dt
		shower.credit += emitDT * shower.rate
		for shower.credit >= 1 && len(shower.particles) < 50 {
			shower.credit--
			offset := func() float32 {
				v := float32(rand.IntN(shower.radius))
				if rand.IntN(2) == 0 {
					v = -v
				}
				return v
			}
			pos := shower.center.Add(render.V3{offset(), shower.height, offset()}).Sub(shower.side.Scale(shower.lateral))
			if x := a.loadSkillEffect(shower.file, 0); x != nil {
				shower.particles = append(shower.particles, skillAreaParticle{pos: pos, velocity: shower.side.Scale(shower.lateral).Add(render.V3{0, -shower.height, 0}), instance: x})
			}
		}
		alive := shower.particles[:0]
		for _, p := range shower.particles {
			if !p.grounded {
				p.pos = p.pos.Add(p.velocity.Scale(dt))
				floor := shower.center[1]
				if a.wv.world != nil && a.wv.world.HasTerrain() {
					floor = a.wv.world.Height(p.pos[0], p.pos[2])
				}
				if p.pos[1] < floor {
					p.grounded = true
					p.pos[1] = floor
					if len(a.wv.impacts) < 32 {
						if x := a.loadSkillEffect(shower.file, 1); x != nil {
							m := render.Identity()
							m[12], m[13], m[14] = p.pos[0], p.pos[1]+1, p.pos[2]
							x.SetTransform(m)
							a.wv.impacts = append(a.wv.impacts, skillImpact{instance: x, fixedWorld: &m})
						}
					}
				}
			} else {
				p.groundAge += dt
			}
			if p.groundAge > 1 {
				continue
			}
			m := render.Identity()
			m[12], m[13], m[14] = p.pos[0], p.pos[1], p.pos[2]
			p.instance.SetTransform(m)
			p.instance.Update(dt)
			alive = append(alive, p)
		}
		clear(shower.particles[len(alive):])
		shower.particles = alive
		if shower.remaining > 0 || len(alive) > 0 {
			rows = append(rows, shower)
		}
	}
	clear(a.wv.areaShowers[len(rows):])
	a.wv.areaShowers = rows
}
