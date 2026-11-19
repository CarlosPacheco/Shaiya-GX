package app

import (
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

type weatherView struct {
	kind        uint8
	blend, time float32
	vertices    []render.Vertex
	indices     []uint32
}

func (a *App) weatherForView() uint8 {
	if a.state == nil || a.wv == nil || a.state.Weather.MapID != a.wv.mapID {
		return 0
	}
	return a.state.Weather.Type
}

func (v *weatherView) update(kind uint8, dt float32) {
	step := min(max(dt, 0), .1) * .75
	switch {
	case kind == 0:
		v.blend = max(v.blend-step, 0)
	case v.kind != kind:
		v.kind = kind
		v.blend = step
	default:
		v.blend = min(v.blend+step, 1)
	}
	if v.blend > 0 {
		v.time += max(dt, 0)
		if v.time > 600 {
			v.time = 0
		}
	}
}

func (v *weatherView) fog(clear render.V3, near, far float32) (render.V3, float32, float32) {
	t := v.blend
	if v.kind == proto.WeatherSnow {
		return clear.Scale(1 - .62*t).Add(render.V3{.70, .74, .77}.Scale(.62 * t)), near * (1 - .68*t), far * (1 - .40*t)
	}
	return clear.Scale(1 - .76*t).Add(render.V3{.35, .41, .48}.Scale(.76 * t)), near * (1 - .68*t), far * (1 - .47*t)
}
func (v *weatherView) sky() (render.V3, float32) {
	t := v.blend
	tint := render.V3{1 - .38*t, 1 - .32*t, 1 - .25*t}
	if v.kind == proto.WeatherSnow {
		tint = render.V3{1 - .13*t, 1 - .11*t, 1 - .07*t}
	}
	return tint, 0x50/255.0 + .32*t
}

// PS107 0x44470f selects rain.tga and snow2008_2.bmp for its middle
// weather settings. Render the local camera volume as one textured batch.
func (v *weatherView) draw(s *render.Scene3D, assets *scene.Assets, eye render.V3) {
	if v.blend <= 0 || assets == nil {
		return
	}
	file := "Effect/DDS/rain.tga"
	count, width, length, speed := 240, float32(.095), float32(.85), float32(28)
	tint := [4]float32{.80, .88, 1, .65 * v.blend}
	snow := v.kind == proto.WeatherSnow
	if snow {
		file = "Effect/DDS/snow2008_2.bmp"
		count, width, length, speed = 80, .28, .28, 2.25
		tint = [4]float32{.95, .97, 1, .85 * v.blend}
	}
	texture := assets.Texture(file)
	if texture == nil {
		return
	}
	right := render.V3{s.View[0], s.View[4], s.View[8]}
	up := render.V3{0, 1, 0}
	if snow {
		up = render.V3{s.View[1], s.View[5], s.View[9]}
	}
	v.vertices = v.vertices[:0]
	v.indices = v.indices[:0]
	for i := range count {
		seed := uint32(i)*747796405 + 2891336453
		rand01 := func() float32 {
			seed = 1664525*seed + 1013904223
			return float32(seed>>8) / 16777216
		}
		x, z := (rand01()-.5)*28, (rand01()-.5)*28
		y := float32(math.Mod(float64(40+rand01()*18-float32(math.Mod(float64(v.time*speed), 18))), 18)) - 7
		if snow {
			x += float32(math.Sin(float64(v.time*1.5+rand01()*9))) * .65
		}
		p := eye.Add(render.V3{x, y, z})
		dx, dy := right.Scale(width), up.Scale(length)
		if !snow {
			dy = dy.Add(right.Scale(-.12))
		}
		base := uint32(len(v.vertices))
		v.vertices = append(v.vertices,
			render.Vertex{Pos: p.Sub(dx).Sub(dy), UV: [2]float32{0, 1}},
			render.Vertex{Pos: p.Add(dx).Sub(dy), UV: [2]float32{1, 1}},
			render.Vertex{Pos: p.Add(dx).Add(dy), UV: [2]float32{1, 0}},
			render.Vertex{Pos: p.Sub(dx).Add(dy), UV: [2]float32{0, 0}},
		)
		v.indices = append(v.indices, base, base+1, base+2, base, base+2, base+3)
	}
	blendDestination := int32(5)
	if snow {
		blendDestination = 1
	} // The native snow bitmap has an opaque black background.
	s.DrawEffectMesh(v.vertices, v.indices, texture, render.Identity(), tint, 4, blendDestination)
}
