package scene

import "github.com/Spelunkern/Shaiya-GX/internal/render"

// NameAnchor follows the crown's cached skin binding, as Phoenix
// player_nameplate.gd. It reuses the pose just drawn, including deaths/movement,
// rather than putting every NPC, monster and player at the same fixed height.
// A pre-culled body may have a visible label; sample only its two crown binds
// on demand instead of using stale part palettes.
func (c *Character) NameAnchor() (render.V3, bool) {
	if c == nil {
		return render.V3{}, false
	}
	p := c.head
	if p == nil {
		for _, part := range c.parts {
			if p == nil || part.max[1] > p.max[1] {
				p = part
			}
		}
	}
	if p == nil {
		return render.V3{}, false
	}
	top := p.min.Add(p.max).Scale(.5)
	top[1] = p.max[1]
	var finals []render.M4
	fresh := c.clip != nil && (!c.poseReady || c.poseClip != c.clip || c.poseTime != c.Time)
	if fresh {
		finals = c.finals()
	}
	var point render.V3
	for i, weight := range [2]float32{p.top.Weight, 1 - p.top.Weight} {
		if weight == 0 {
			continue
		}
		bone := int(p.top.Bone0)
		if i == 1 {
			bone = int(p.top.Bone1)
		}
		var pose render.M4
		if fresh {
			if bone < 0 || bone >= len(p.bones) {
				return render.V3{}, false
			}
			final := render.Identity()
			if bone < len(finals) {
				final = finals[bone]
			}
			pose = p.bones[bone].Mul(final)
		} else {
			if bone < 0 || bone >= len(p.pose) {
				return render.V3{}, false
			}
			pose = p.pose[bone]
		}
		m := pose.Mul(c.World)
		for axis := range 3 {
			point[axis] += weight * (top[0]*m[axis] + top[1]*m[4+axis] + top[2]*m[8+axis] + m[12+axis])
		}
	}
	point[1] += .15
	return point, true
}
