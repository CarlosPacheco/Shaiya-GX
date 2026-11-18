package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

// placedField is a field left on the ground (Fire Bush's curtain of fire): its
// area effect is shown at the spot, again whenever it ends, until the time the
// server announced runs out.
type placedField struct {
	file      string
	sequence  int
	at        render.V3
	remaining float32
	instance  *scene.Effect
}

func (a *App) placeField(f proto.Field) {
	if a.wv == nil || len(a.wv.fields) >= 16 {
		return
	}
	file := effect.AreaLibrary(f.Skill)
	if file == "" || f.Seconds == 0 {
		return
	}
	a.wv.fields = append(a.wv.fields, placedField{file: file, sequence: int(f.Level) - 1, at: render.V3{f.X, f.Y, f.Z}, remaining: float32(f.Seconds)})
	a.playSound(effect.ImpactSound(file), f.X, f.Z)
}

func (a *App) updateFields(dt float32) {
	kept := a.wv.fields[:0]
	for _, f := range a.wv.fields {
		f.remaining -= dt
		if f.remaining <= 0 {
			if f.instance != nil {
				f.instance.Stop()
			}
			continue
		}
		if (f.instance == nil || f.instance.Done()) && len(a.wv.impacts) < 32 {
			if x := a.loadSkillEffect(f.file, f.sequence); x != nil {
				m := render.Identity()
				m[12], m[13], m[14] = f.at[0], f.at[1], f.at[2]
				if a.wv.world != nil && a.wv.world.HasTerrain() {
					m[13] = a.wv.world.Height(f.at[0], f.at[2])
				}
				x.SetTransform(m)
				a.wv.impacts = append(a.wv.impacts, skillImpact{instance: x, authoredLifetime: true, fixedWorld: &m})
				f.instance = x
			}
		}
		kept = append(kept, f)
	}
	clear(a.wv.fields[len(kept):])
	a.wv.fields = kept
}
