package scene

import (
	"math"
	"slices"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// A global bone transform maps a point of length r to length <= scale*r+travel.
// These bounds cover the entire clip, not just its keyframe poses.
type boneReach struct{ scale, travel float64 }

type envelopeAttachment struct {
	part  *Part
	bone  int
	extra render.M4
}

type characterEnvelope struct {
	clip                  *Clip
	parts                 []*skinPart
	attachments           []envelopeAttachment
	mounted, ready, valid bool
	radius                float64
}

// Keep all intermediate float32 matrix products away from overflow as well
// as rejecting NaN/Inf. Values beyond this range use exact culling.
func finiteBound(x float64) bool { return x >= -1e12 && x <= 1e12 }

// makeReach uses the triangle inequality through the same parent order as
// Finals. Translation lerps stay in the convex hull of their endpoints. Slerp
// normalizes interior rotations; raw endpoint quaternions can have small scale
// error. For q with squared norm N, ||QuatMatrix(q)|| <= max(1,2*N-1).
// Unsupported/malformed clips leave pre-culling disabled.
func (c *Clip) makeReach() []boneReach {
	if c.ani == nil || c.ani.Start > 1<<24 || c.ani.End > 1<<24 || len(c.restLocal) != len(c.ani.Bones) {
		return nil
	}
	out := make([]boneReach, len(c.ani.Bones))
	for i, bone := range c.ani.Bones {
		q, t := render.MatrixQuat(c.restLocal[i])
		rotationScale := func(q [4]float32) float64 {
			var norm float64
			for _, v := range q {
				norm += float64(v) * float64(v)
			}
			if !finiteBound(norm) || math.Abs(norm-1) > .001 {
				return math.Inf(1)
			}
			return math.Max(1, 2*norm-1) + .00001
		}
		scale := rotationScale(q)
		if len(bone.Rot) > 0 {
			scale = 1
			for j, k := range bone.Rot {
				if k.Frame > 1<<24 {
					return nil
				}
				if j > 0 && float32(k.Frame) <= float32(bone.Rot[j-1].Frame) {
					return nil
				}
				scale = math.Max(scale, rotationScale(k.Q))
			}
		}
		length := func(v [3]float32) float64 {
			return math.Sqrt(float64(v[0])*float64(v[0]) + float64(v[1])*float64(v[1]) + float64(v[2])*float64(v[2]))
		}
		travel := length(t)
		if len(bone.Trans) > 0 {
			travel = 0
			for j, k := range bone.Trans {
				if k.Frame > 1<<24 {
					return nil
				}
				if j > 0 && float32(k.Frame) <= float32(bone.Trans[j-1].Frame) {
					return nil
				}
				travel = math.Max(travel, length(k.T))
			}
		}
		travel = travel*1.00001 + .00001
		if p := bone.Parent; p >= 0 && int(p) < i {
			travel = travel*out[p].scale + out[p].travel
			scale *= out[p].scale
		}
		if !finiteBound(scale) || !finiteBound(travel) {
			return nil
		}
		out[i] = boneReach{scale, travel}
	}
	return out
}

func affineEnvelope(m render.M4) bool {
	for _, v := range m {
		if !finiteBound(float64(v)) {
			return false
		}
	}
	return m[3] == 0 && m[7] == 0 && m[11] == 0 && m[15] == 1
}

// boxRadius encloses all corners after the inverse bind/equipment transform.
func boxRadius(lo, hi render.V3, m render.M4) float64 {
	if !affineEnvelope(m) {
		return math.Inf(1)
	}
	radius := float64(0)
	for corner := range 8 {
		p := lo
		for axis := range 3 {
			if !finiteBound(float64(lo[axis])) || !finiteBound(float64(hi[axis])) || lo[axis] > hi[axis] {
				return math.Inf(1)
			}
			if corner&(1<<axis) != 0 {
				p[axis] = hi[axis]
			}
		}
		var squared float64
		for axis := range 3 {
			v := math.Abs(float64(p[0])*float64(m[axis])) + math.Abs(float64(p[1])*float64(m[4+axis])) + math.Abs(float64(p[2])*float64(m[8+axis])) + math.Abs(float64(m[12+axis]))
			squared += v * v
		}
		radius = math.Max(radius, math.Sqrt(squared))
	}
	return radius*1.00001 + .00001
}

func (c *Character) envelopeRadius() (float64, bool) {
	if c == nil || c.clip == nil || c.clip.ani == nil || len(c.clip.reach) != len(c.clip.ani.Bones) || len(c.clip.reach) == 0 {
		return 0, false
	}
	e := &c.envelope
	same := e.ready && e.clip == c.clip && e.mounted == c.Mounted && slices.Equal(e.parts, c.parts) && len(e.attachments) == len(c.attachments)
	for i, a := range c.attachments {
		if same && e.attachments[i] != (envelopeAttachment{a.part, a.bone, a.extra}) {
			same = false
		}
	}
	if same {
		return e.radius, e.valid
	}
	e.ready, e.valid, e.clip, e.mounted = true, false, c.clip, c.Mounted
	e.parts = append(e.parts[:0], c.parts...)
	e.attachments = e.attachments[:0]
	for _, a := range c.attachments {
		e.attachments = append(e.attachments, envelopeAttachment{a.part, a.bone, a.extra})
	}
	radius, has := float64(0), false
	for _, p := range c.parts {
		if p == nil || !p.bounded {
			return 0, false
		}
		for i, b := range p.bounds {
			if !b.used {
				continue
			}
			bind, reach := render.Identity(), boneReach{scale: 1}
			if len(p.bones) > 0 {
				if i >= len(p.bones) {
					return 0, false
				}
				bind = p.bones[i]
				if i < len(c.clip.reach) {
					reach = c.clip.reach[i]
				}
			}
			radius = math.Max(radius, boxRadius(b.min, b.max, bind)*reach.scale+reach.travel)
			has = true
		}
	}
	if !c.Mounted {
		for _, a := range c.attachments {
			if a.bone < 0 || a.bone >= len(c.clip.reach) {
				continue
			}
			if a.part == nil || !a.part.bounded {
				return 0, false
			}
			reach := c.clip.reach[a.bone]
			radius = math.Max(radius, boxRadius(a.part.boundMin, a.part.boundMax, a.extra)*reach.scale+reach.travel)
			has = true
		}
	}
	// Outward slack also covers float32 matrix products and WorldBounds' .01 pad.
	e.radius = radius*1.0001 + .02
	e.valid = has && finiteBound(e.radius)
	return e.radius, e.valid
}

// coarseBodyBounds encloses the same geometry as WorldBounds, including a
// cloak's skinned shoulders. The separate cloth mesh enters occlusion at draw.
// Transforming a sphere by a general
// affine world matrix gives per-axis extents r*length(column), including scale,
// reflection and shear. An uncertain bound always uses the exact existing path.
func (c *Character) coarseBodyBounds() (render.V3, render.V3, bool) {
	// The incoming clip's bounds do not cover the outgoing blend pose.
	if c == nil || c.blendLeft > 0 {
		return render.V3{}, render.V3{}, false
	}
	radius, valid := c.envelopeRadius()
	if !valid || !finiteBound(float64(c.Time)) || !affineEnvelope(c.World) {
		return render.V3{}, render.V3{}, false
	}
	lo, hi := render.V3{}, render.V3{}
	for axis := range 3 {
		x, y, z := float64(c.World[axis]), float64(c.World[4+axis]), float64(c.World[8+axis])
		extent := radius*math.Sqrt(x*x+y*y+z*z) + .02
		a, b := float64(c.World[12+axis])-extent, float64(c.World[12+axis])+extent
		lo[axis], hi[axis] = float32(a), float32(b)
		// One outward float32 step avoids rounding the enclosing box inward.
		lo[axis] = math.Nextafter32(lo[axis], float32(math.Inf(-1)))
		hi[axis] = math.Nextafter32(hi[axis], float32(math.Inf(1)))
		if !finiteBound(float64(lo[axis])) || !finiteBound(float64(hi[axis])) {
			return render.V3{}, render.V3{}, false
		}
	}
	return lo, hi, true
}
