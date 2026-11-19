package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

type weatherView struct {
	kind         uint8
	blend        float32
	spawnElapsed float64
	random       uint32
	drops        []weatherDrop
	vertices     []render.Vertex
	indices      []uint32
}

type weatherDrop struct {
	position, velocity render.V3
	age                float32
}

type precipitation struct {
	texture                            string
	interval                           float64
	life, extent                       float32
	fall, halfWidth, halfHeight, shade float32
	destination                        int32
}

// Timing and motion come from PS107's middle weather variants (0x444738 and 0x444792).
// The emitter uses integer millisecond intervals and saves world-space birth
// positions. The 0x449e24 pass places its origin ahead of the camera.
var normalRain = precipitation{
	texture: "Effect/DDS/rain.tga", interval: (1000 / 240) / 1000.0, life: 1.5,
	extent: 250, fall: -50, halfWidth: 1.25, halfHeight: .70, shade: 175.0 / 255, destination: 5,
}
var normalSnow = precipitation{
	texture: "Effect/DDS/snow2008_2.bmp", interval: (1000 / 80) / 1000.0, life: 7,
	extent: 400, fall: -2.5, halfWidth: 200.0 / 127, halfHeight: 200.0 / 127, shade: 1, destination: 1,
}

func (v *weatherView) precipitation() precipitation {
	if v.kind == proto.WeatherSnow {
		return normalSnow
	}
	return normalRain
}

func (a *App) weatherForView() uint8 {
	if a.state == nil || a.wv == nil || a.state.Weather.MapID != a.wv.mapID {
		return 0
	}
	return a.state.Weather.Type
}

func (v *weatherView) update(kind uint8, dt float32, origin render.V3) {
	dt = min(max(dt, 0), .1)
	step := dt * .2 // PS107 completes the sky transition in five seconds.
	if kind != 0 && v.kind != kind {
		v.kind = kind
		v.drops = v.drops[:0]
		v.spawnElapsed = 0
	}
	if kind == 0 {
		v.blend = max(v.blend-step, 0)
	} else {
		v.blend = min(v.blend+step, 1)
	}
	p := v.precipitation()
	active := v.drops[:0]
	for _, drop := range v.drops {
		drop.age += dt
		if drop.age < p.life {
			active = append(active, drop)
		}
	}
	v.drops = active
	if kind == 0 {
		return
	}
	v.spawnElapsed += float64(dt)
	for v.spawnElapsed >= p.interval {
		v.spawnElapsed -= p.interval
		// Shape 9 distributes each birth around the emitter, with a
		// narrower vertical range. The two horizontal components drift
		// independently of the view after birth.
		offset := render.V3{v.spread(p.extent * .1), v.spread(p.extent * .01), v.spread(p.extent * .1)}
		velocity := render.V3{v.spread(1), p.fall, v.spread(1)}
		v.drops = append(v.drops, weatherDrop{origin.Add(offset), velocity, float32(v.spawnElapsed)})
	}
}

func (v *weatherView) spread(radius float32) float32 {
	v.random = 1664525*v.random + 1013904223
	return (float32(v.random>>8)/8388608 - 1) * radius
}

func (v *weatherView) fog(clear render.V3, near, far float32) (render.V3, float32, float32) {
	t := v.blend
	if v.kind == proto.WeatherSnow {
		return clear.Scale(1 - .62*t).Add(render.V3{.70, .74, .77}.Scale(.62 * t)), near * (1 - .68*t), far * (1 - .40*t)
	}
	return clear.Scale(1 - t).Add(render.V3{.43, .43, .43}.Scale(t)), near * (1 - .68*t), far * (1 - .47*t)
}

// Render the emitter's billboards as one textured batch.
func (v *weatherView) draw(s *render.Scene3D, assets *scene.Assets, eye render.V3) {
	if len(v.drops) == 0 || assets == nil {
		return
	}
	p := v.precipitation()
	texture := assets.Texture(p.texture)
	if texture == nil {
		return
	}
	right := render.V3{s.View[0], s.View[4], s.View[8]}
	up := render.V3{s.View[1], s.View[5], s.View[9]}
	dx, dy := right.Scale(p.halfWidth), up.Scale(p.halfHeight)
	v.vertices = v.vertices[:0]
	v.indices = v.indices[:0]
	minDistance := float32(1)
	if v.kind == proto.WeatherRain {
		minDistance = 2.5 // Keep elongated drops away from the near plane.
	}
	for _, drop := range v.drops {
		position := drop.position.Add(drop.velocity.Scale(drop.age))
		delta := position.Sub(eye)
		if delta.Dot(delta) < minDistance*minDistance {
			continue
		}
		base := uint32(len(v.vertices))
		v.vertices = append(v.vertices,
			render.Vertex{Pos: position.Sub(dx).Sub(dy), UV: [2]float32{0, 1}},
			render.Vertex{Pos: position.Add(dx).Sub(dy), UV: [2]float32{1, 1}},
			render.Vertex{Pos: position.Add(dx).Add(dy), UV: [2]float32{1, 0}},
			render.Vertex{Pos: position.Sub(dx).Add(dy), UV: [2]float32{0, 0}},
		)
		v.indices = append(v.indices, base, base+1, base+2, base, base+2, base+3)
	}
	s.DrawEffectMesh(v.vertices, v.indices, texture, render.Identity(),
		[4]float32{p.shade, p.shade, p.shade, 1}, 4, p.destination)
}
