package scene

import (
	"fmt"
	"math"
	"math/rand/v2"
	"path"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type effectParticle struct {
	curve          velocityCurve
	pos, velocity  render.V3
	age, life      float32
	staticTime     float32
	sizeSeed       float32
	staticSizeSeed float32
	angle, speed   float32
	tilt           effectTilt
	frame          float32
}
type effectEmitter struct {
	definition              *effect.Emitter
	geometry                *effectGeometry
	textures                []*render.Texture
	textureFrame            int
	textureClock            float32
	starts, time, remainder float32
	particles               []effectParticle
	// path is the .3DE of an emitter that does not draw it but moves its births
	// along it (Unk10, "motion path"): a particle is born where the mesh's last
	// vertex is at that moment, so a ring that unrolls over a second makes a
	// ring of flames.
	path *effectGeometry
}

// Effect owns one live emitter group; parsed templates and its GPU quad are shared.
// Skipped emitters are reported rather than replaced with invented geometry.
type Effect struct {
	emitters       []effectEmitter
	quad           *render.Mesh
	mirrorQuad     *render.Mesh
	Origin         render.V3
	Time, step     float32
	Skipped        int
	stopped        bool
	basis          render.M4
	localParticles bool
	meteor         bool
	meteorWorld    render.M4
	particleWorld  render.M4
	batchWorld     []render.M4
	batchColor     [][4]float32
	batchFrames    []float32
	batchPoses     []render.EffectPose
	draws          []effectDraw
	geometryCache  *render.EffectGeometryCache
}

type effectTemplate struct {
	emitters []effectEmitter
	skipped  int
}

// Native 0x55ec16 advances the emitter's texture clock once per update,
// resetting its remainder and looping according to the +0x100 flag.
func (e *effectEmitter) advanceTexture(dt float32) {
	if len(e.textures) <= 1 {
		return
	}
	e.textureClock += dt
	if e.textureClock <= e.definition.FrameDelay {
		return
	}
	e.textureClock = 0
	if e.textureFrame+1 < len(e.textures) {
		e.textureFrame++
	} else if e.definition.Unk08 != 0 {
		e.textureFrame = 0
	}
}

func LoadEffect(a *Assets, file string, sequence int) (*Effect, error) {
	key := fmt.Sprintf("%s:%d", strings.ToLower(file), sequence)
	if t := a.effectTemplates[key]; t != nil {
		return newEffect(a, t), nil
	}
	b, err := a.Arc.ReadFile(file)
	if err != nil {
		return nil, err
	}
	l, err := effect.ParseEFT(b)
	if err != nil {
		return nil, err
	}
	return loadEffectLibrary(a, file, sequence, l)
}

// A world can contain hundreds of placements from one library. Reuse the
// parsed library while retaining the existing bounded sequence templates.
func loadEffectLibrary(a *Assets, file string, sequence int, l *effect.Library) (*Effect, error) {
	key := fmt.Sprintf("%s:%d", strings.ToLower(file), sequence)
	if t := a.effectTemplates[key]; t != nil {
		return newEffect(a, t), nil
	}
	if l.Format != "EFT" {
		return nil, fmt.Errorf("EFT: runtime for %s not verified", l.Format)
	}
	if sequence >= len(l.Sequences) {
		sequence = len(l.Sequences) - 1
	}
	if sequence < 0 {
		return nil, fmt.Errorf("EFT: sequence %d out of range", sequence)
	}
	x := &Effect{basis: render.Identity(), geometryCache: &a.gpuEffects}
	for _, r := range l.Sequences[sequence].Records {
		if r.Effect < 0 || int(r.Effect) >= len(l.Effects) {
			return nil, fmt.Errorf("EFT: effect %d out of range", r.Effect)
		}
		e := &l.Effects[r.Effect]
		if !e.LinearSupported() {
			x.Skipped++
			continue
		}
		var geometry, route *effectGeometry
		if e.Mesh >= 0 && !e.UsesMesh() && int(e.Mesh) < len(l.Meshes) {
			if mesh, err := a.loadEffectGeometry(l.Meshes[e.Mesh]); err == nil && len(mesh.source.Vertices) > 0 {
				route = mesh
			}
		}
		if e.UsesMesh() {
			if int(e.Mesh) >= len(l.Meshes) {
				return nil, fmt.Errorf("EFT: mesh %d out of range", e.Mesh)
			}
			var err error
			geometry, err = a.loadEffectGeometry(l.Meshes[e.Mesh])
			if err != nil {
				return nil, err
			}
		}
		textures := make([]*render.Texture, 0, len(e.Textures))
		for _, id := range e.Textures {
			if id < 0 || int(id) >= len(l.Textures) {
				return nil, fmt.Errorf("EFT: texture %d out of range", id)
			}
			name := path.Base(strings.ReplaceAll(l.Textures[id], `\`, "/"))
			t := a.Texture("Effect/DDS/" + name)
			if t == nil {
				return nil, fmt.Errorf("EFT: missing texture %s", name)
			}
			textures = append(textures, t)
		}
		x.emitters = append(x.emitters, effectEmitter{definition: e, geometry: geometry, textures: textures, starts: r.Time, path: route})
	}
	if len(x.emitters) == 0 {
		return nil, fmt.Errorf("EFT: no supported linear emitters (%d skipped)", x.Skipped)
	}
	if a.effectQuad == nil {
		a.effectQuad = render.NewMesh([]render.Vertex{{Pos: render.V3{-.5, -.5, 0}, UV: [2]float32{0, 1}}, {Pos: render.V3{.5, -.5, 0}, UV: [2]float32{1, 1}}, {Pos: render.V3{.5, .5, 0}, UV: [2]float32{1, 0}}, {Pos: render.V3{-.5, .5, 0}, UV: [2]float32{0, 0}}}, []uint32{0, 1, 2, 0, 2, 3})
	}
	if a.effectMirrorQuad == nil {
		vertices, indices := mirroredEffectQuad()
		a.effectMirrorQuad = render.NewMesh(vertices, indices)
	}
	x.quad = a.effectQuad
	x.mirrorQuad = a.effectMirrorQuad
	if a.effectTemplates == nil {
		a.effectTemplates = map[string]*effectTemplate{}
	}
	if len(a.effectOrder) >= 32 {
		delete(a.effectTemplates, a.effectOrder[0])
		a.effectOrder = a.effectOrder[1:]
	}
	a.effectOrder = append(a.effectOrder, key)
	a.effectTemplates[key] = &effectTemplate{emitters: append([]effectEmitter(nil), x.emitters...), skipped: x.Skipped}
	return x, nil
}

func (a *Assets) loadEffectGeometry(filename string) (*effectGeometry, error) {
	name := path.Base(strings.ReplaceAll(filename, `\`, "/"))
	key := strings.ToLower(name)
	if g := a.effectMeshes[key]; g != nil {
		return g, nil
	}
	raw, err := a.Arc.ReadFile("Effect/3DE/" + name)
	if err != nil {
		return nil, err
	}
	mesh, err := effect.Parse3DE(raw)
	if err != nil {
		return nil, fmt.Errorf("EFT mesh %s: %w", name, err)
	}
	g := newEffectGeometry(mesh)
	if len(a.effectMeshes) < 128 {
		if a.effectMeshes == nil {
			a.effectMeshes = make(map[string]*effectGeometry)
		}
		a.effectMeshes[key] = g
	}
	return g, nil
}

func newEffect(a *Assets, t *effectTemplate) *Effect {
	return &Effect{emitters: append([]effectEmitter(nil), t.emitters...), quad: a.effectQuad, mirrorQuad: a.effectMirrorQuad, Skipped: t.skipped, basis: render.Identity(), geometryCache: &a.gpuEffects}
}

// motionPathOffset is where an emitter's births are placed at a given time of
// the emitter: the last vertex of its path mesh, sampled at 30 keys a second
// and wrapping at the mesh's last key.
func (g *effectGeometry) motionPathOffset(seconds float32) render.V3 {
	last := len(g.source.Vertices) - 1
	if len(g.source.Frames) == 0 {
		return render.V3(g.source.Vertices[last].Position)
	}
	a, b, t := g.framesAt(float32(math.Mod(float64(seconds)*30, float64(g.source.MaxKeyframe)+1)))
	return interpolate3DEPosition(g.source.Frames[a].Vertices[last].Position, g.source.Frames[b].Vertices[last].Position, t)
}

func between(a, b float32) float32 {
	return float32(float64(a) + (float64(b)-float64(a))*rand.Float64())
}

func (x *Effect) Update(dt float32) {
	// Bound catch-up after a pause; this is visual state, never gameplay.
	x.step += min(max(dt, 0), .25)
	for x.step >= 1.0/60 {
		x.step -= 1.0 / 60
		x.Time += 1.0 / 60
		for i := range x.emitters {
			e := &x.emitters[i]
			d := e.definition
			p := d.Parameters()
			if x.Time < e.starts {
				continue
			}
			previousTime := e.time
			e.time += 1.0 / 60
			e.advanceTexture(1.0 / 60)
			var orbitSin, orbitCos float32
			if d.Unk06 != 0 && len(e.particles) != 0 {
				orbitSin, orbitCos = float32(math.Sin(float64(d.Unk17)/60)), float32(math.Cos(float64(d.Unk17)/60))
			}
			attractor := render.V3(d.UnkVec6)
			if d.Unk21 != 0 && e.geometry == nil {
				attractor = effectPoint(attractor, x.basis).Add(x.Origin)
			}
			alive := e.particles[:0]
			for _, v := range e.particles {
				if v.life == 0 {
					if x.stopped {
						continue
					}
					v.staticTime += 1.0 / 60
					if e.geometry == nil {
						v.pos = render.V3(d.Position)
						for axis := range v.pos {
							v.pos[axis] += between(-p.PositionSpread[axis], p.PositionSpread[axis])
						}
						v.pos = effectPoint(v.pos, x.basis).Add(x.Origin)
					}
				} else {
					v.age += 1.0 / 60
				}
				if v.life > 0 && v.age > v.life {
					continue
				}
				for axis, flag := range []int32{d.Unk01, d.Unk02, d.Unk03} {
					if flag != 0 {
						v.velocity[axis] = between(p.VelocityMin[axis], p.VelocityMax[axis])
					}
				}
				if d.Unk06 == 0 {
					v.pos = v.pos.Add(v.velocity.Scale(1.0 / 60))
				} else {
					v.velocity = orbitStep(v.velocity, d.Unk06, orbitSin, orbitCos)
					for axis := range v.velocity {
						v.velocity[axis] += between(p.VelocityMin[axis], p.VelocityMax[axis]) / 60
					}
				}
				if len(d.Opacity) > 0 {
					acceleration := v.curve.step(d.Opacity, v.age, 1.0/60)
					v.velocity = v.velocity.Add(v.velocity.Scale(acceleration / 60))
				}
				if d.Unk21 != 0 {
					v.velocity = v.velocity.Add(attractor.Sub(v.pos).Norm().Scale(d.RotationMin / 60))
				}
				if d.Unk20 != 0 {
					v.velocity = v.velocity.Add(render.V3(d.Trembling).Scale(1.0 / 60))
				}
				v.angle += v.speed / 60
				if e.geometry != nil {
					v.frame = advanceMeshFrame(v.frame, e.geometry.source.MaxKeyframe, d.Unk08 != 0, 1.0/60)
				}
				alive = append(alive, v)
			}
			e.particles = alive
			if x.stopped || d.Loop == 0 && previousTime >= p.Duration {
				continue
			}
			count := 0
			if p.RateMin == 1 && p.RateMax == 1 {
				if e.time >= p.Duration {
					count = 1
				}
			} else {
				if d.Loop == 0 && e.time > p.Duration {
					continue
				}
				e.remainder += min(max(0, between(p.RateMin, p.RateMax))/60, 200)
				count = int(e.remainder)
				e.remainder -= float32(count)
			}
			count = min(count, 200-len(e.particles))
			for n := 0; n < count; n++ {
				v := effectParticle{pos: render.V3(d.Position), life: between(p.LifeMin, p.LifeMax), sizeSeed: rand.Float32()}
				if v.life == 0 && len(d.Sub3) > 1 {
					v.staticSizeSeed = rand.Float32()
				}
				low, high := float32(d.Unk31), float32(d.Unk32)
				if high < low {
					if rand.IntN(2) == 0 {
						low = 0
					} else {
						high = 360
					}
				}
				v.tilt = newEffectTilt(d.Unk30, between(low, high)*math.Pi/180)
				if d.RotationEnabled != 0 {
					v.speed = d.RotationMax
					if d.RotationRandom != 0 {
						v.speed = between(0, v.speed)
					}
				}
				for axis := range v.pos {
					v.pos[axis] += between(-p.PositionSpread[axis], p.PositionSpread[axis])
					v.velocity[axis] = between(p.VelocityMin[axis], p.VelocityMax[axis])
				}
				if e.path != nil {
					v.pos = v.pos.Add(e.path.motionPathOffset(e.time))
				}
				if !(x.meteor && e.geometry != nil) {
					v.pos = effectPoint(v.pos, x.basis).Add(x.Origin)
				}
				e.particles = append(e.particles, v)
			}
		}
	}
}

func particleSize(e *effect.Emitter, age, seed float32) float32 {
	if len(e.Sub3) == 0 {
		return .3
	}
	value := func(f effect.Sub3Frame) float32 { return f.Unk1 + (f.Unk2-f.Unk1)*seed }
	for i := 1; i < len(e.Sub3); i++ {
		a, b := e.Sub3[i-1], e.Sub3[i]
		if age < b.Time {
			if b.Time <= a.Time {
				return value(b)
			}
			t := max(0, (age-a.Time)/(b.Time-a.Time))
			return value(a) + (value(b)-value(a))*t
		}
	}
	return value(e.Sub3[len(e.Sub3)-1])
}

func particleColor(e *effect.Emitter, age, life float32) [4]float32 {
	if len(e.Colors) == 0 {
		v := min(age/life, 1)
		return [4]float32{v, v, v, v}
	}
	if age < e.Colors[0].Time {
		return [4]float32{}
	}
	for i := 1; i < len(e.Colors); i++ {
		a, b := e.Colors[i-1], e.Colors[i]
		if age < b.Time && b.Time > a.Time {
			t := (age - a.Time) / (b.Time - a.Time)
			var c [4]float32
			for j := range c {
				c[j] = a.RGBA[j] + (b.RGBA[j]-a.RGBA[j])*t
			}
			return c
		}
	}
	return e.Colors[len(e.Colors)-1].RGBA
}

// For a zero-life particle native age remains zero. Its first color segment
// starts on the first update and its slope continues while the effect exists.
func staticParticleColor(e *effect.Emitter, elapsed float32) [4]float32 {
	if elapsed <= 0 {
		return [4]float32{}
	}
	// Native default color slopes are 1/life. For life zero they become
	// positive infinity, which the draw path clamps to white on the first tick.
	if len(e.Colors) == 0 {
		return [4]float32{1, 1, 1, 1}
	}
	if len(e.Colors) < 2 {
		return [4]float32{}
	}
	a, b := e.Colors[0], e.Colors[1]
	t := elapsed / (b.Time - a.Time)
	var c [4]float32
	for i := range c {
		c[i] = a.RGBA[i] + (b.RGBA[i]-a.RGBA[i])*t
	}
	return c
}

// A zero-life particle never reaches the second size key in native time.
// The first key chooses its initial size and the second supplies the target.
func staticParticleSize(e *effect.Emitter, elapsed, initialSeed, targetSeed float32) float32 {
	if len(e.Sub3) == 0 {
		return .3
	}
	a := e.Sub3[0]
	start := a.Unk1 + (a.Unk2-a.Unk1)*initialSeed
	if len(e.Sub3) < 2 || elapsed <= 0 {
		return start
	}
	b := e.Sub3[1]
	target := b.Unk1 + (b.Unk2-b.Unk1)*targetSeed
	t := min(elapsed/(b.Time-a.Time), 1)
	return start + (target-start)*t
}

func (x *Effect) Draw(s *render.Scene3D) {
	s.Begin()
	x.drawInPass(s)
	s.End()
}

// A prepared draw refers to actor-owned placement/color arrays and immutable
// material and frame references. Shared 3DE scratch is sampled at submission.
type effectDraw struct {
	mesh                *render.Mesh
	geometry            *effectGeometry
	texture             *render.Texture
	start, end, poses   int
	source, destination int32
	clamp               bool
}

func (x *Effect) drawInPass(s *render.Scene3D) {
	x.prepareDraw(s.View, s.ActiveOcclusion())
	x.submitDraw(s)
}

func (x *Effect) prepareDraw(view render.M4, occlusion *render.TerrainOcclusion) {
	x.batchWorld, x.batchColor = x.batchWorld[:0], x.batchColor[:0]
	x.batchFrames, x.batchPoses = x.batchFrames[:0], x.batchPoses[:0]
	clear(x.draws)
	x.draws = x.draws[:0]
	for _, e := range x.emitters {
		if len(e.textures) == 0 || len(e.particles) == 0 {
			continue
		}
		p := e.definition.Parameters()
		draw := effectDraw{mesh: x.quad, geometry: e.geometry, texture: e.textures[e.textureFrame], start: len(x.batchWorld), poses: len(x.batchPoses), source: p.SourceBlend, destination: p.DestinationBlend, clamp: e.definition.Unk29 != 0}
		if draw.clamp {
			draw.mesh = x.mirrorQuad
		}
		placement := x.basis
		if x.localParticles {
			placement = x.particleWorld
		}
		if x.meteor && e.geometry != nil {
			placement = x.meteorWorld
		}
		basis := effectBaseBasis(e.definition.BaseAxis, view, placement)
		for _, v := range e.particles {
			if g := e.geometry; g != nil {
				frame := v.frame
				if len(g.source.Frames) <= 1 {
					frame = 0
				}
				var pose render.EffectPose
				if len(g.source.Frames) != 0 {
					a, b, t := g.framesAt(frame)
					pose = render.EffectPose{float32(a * len(g.vertices)), float32(b * len(g.vertices)), t}
				}
				x.batchFrames = append(x.batchFrames, frame)
				x.batchPoses = append(x.batchPoses, pose)
			}
			var size float32
			var color [4]float32
			if v.life == 0 {
				size = staticParticleSize(e.definition, v.staticTime, v.sizeSeed, v.staticSizeSeed)
				color = staticParticleColor(e.definition, v.staticTime)
			} else {
				size = particleSize(e.definition, v.age, v.sizeSeed)
				color = particleColor(e.definition, v.age, v.life)
			}
			m := basis
			for row := range 3 {
				m[row] *= size
				m[4+row] *= size
				m[8+row] *= size
			}
			m = particleRotation(e.definition, v.tilt, v.angle, m, e.geometry != nil)
			pos := x.particlePosition(v, e.definition)
			m[12], m[13], m[14] = pos[0], pos[1], pos[2]
			for i, c := range color {
				color[i] = float32(int(min(max(c, 0), 1)*255)) / 255
			}
			x.batchWorld = append(x.batchWorld, m)
			x.batchColor = append(x.batchColor, color)
		}
		draw.end = len(x.batchWorld)
		if e.geometry == nil && occlusion != nil {
			draw.end = draw.start + occlusion.CullEffectInstances(draw.mesh, x.batchWorld[draw.start:], x.batchColor[draw.start:])
			x.batchWorld, x.batchColor = x.batchWorld[:draw.end], x.batchColor[:draw.end]
			if draw.end == draw.start {
				continue
			}
		}
		x.draws = append(x.draws, draw)
	}
}

func (x *Effect) submitDraw(s *render.Scene3D) {
	for _, draw := range x.draws {
		worlds, colors := x.batchWorld[draw.start:draw.end], x.batchColor[draw.start:draw.end]
		if g := draw.geometry; g != nil {
			if cached := x.geometryCache.Geometry(g.id, g.vertices, g.indices, len(g.vertices)*len(g.source.Frames), g.keyVertexData); cached != nil {
				poses := x.batchPoses[draw.poses : draw.poses+len(worlds)]
				s.DrawEffectGeometry(cached, draw.texture, worlds, colors, poses, draw.source, draw.destination)
			} else {
				frames := x.batchFrames[draw.poses : draw.poses+len(worlds)]
				for start := 0; start < len(frames); {
					end := start + 1
					for end < len(frames) && frames[start] == frames[end] {
						end++
					}
					s.DrawEffectMeshBatch(g.verticesAt(frames[start]), g.indices, draw.texture, worlds[start:end], colors[start:end], draw.source, draw.destination)
					start = end
				}
			}
			continue
		}
		s.DrawVisibleEffectBatch(draw.mesh, draw.texture, worlds, colors, draw.source, draw.destination, draw.clamp)
	}
}

func (x *Effect) particlePosition(v effectParticle, d *effect.Emitter) render.V3 {
	pos := v.pos
	if d.Unk06 != 0 {
		pos = pos.Add(v.velocity)
	}
	if x.localParticles {
		pos = transformPoint(pos, x.particleWorld)
	} else if x.meteor && d.UsesMesh() {
		pos = transformPoint(pos, x.meteorWorld)
	}
	return pos
}

// 0x55de50/0x55d270 rotate the position offset, then add a random
// per-axis drift. The base position remains fixed for orbit modes 1..3.
func orbitStep(v render.V3, mode int32, s, c float32) render.V3 {
	switch mode {
	case 1:
		v[0], v[2] = c*v[0]-s*v[2], c*v[2]+s*v[0]
	case 2:
		v[0], v[1] = c*v[0]+s*v[1], c*v[1]-s*v[0]
	case 3:
		v[1], v[2] = c*v[1]+s*v[2], c*v[2]-s*v[1]
	}
	return v
}

// Game.exe 0x55b1f0 selects camera, horizontal, upright or effect-world
// matrices. 0x447ef0 constructs the two constrained bases from the camera.
func effectBaseBasis(axis int32, view, world render.M4) render.M4 {
	switch axis {
	case 1:
		return effectRotation(1, math.Pi/2)
	case 2:
		return effectRotation(2, float32(math.Atan2(float64(view[2]), float64(view[10]))))
	case 3:
		return world
	default:
		m := render.Identity()
		for row := 0; row < 3; row++ {
			m[row] = view[row*4]
			m[4+row] = view[1+row*4]
			m[8+row] = view[2+row*4]
		}
		return m
	}
}

func effectRotation(axis int32, angle float32) render.M4 {
	if angle == 0 || axis < 1 || axis > 3 {
		return render.Identity()
	}
	q := [4]float32{0, 0, 0, float32(math.Cos(float64(angle) * .5))}
	q[axis-1] = float32(math.Sin(float64(angle) * .5))
	return render.QuatMatrix(q, render.V3{})
}
func effectPoint(p render.V3, m render.M4) render.V3 {
	return render.V3{p[0]*m[0] + p[1]*m[4] + p[2]*m[8], p[0]*m[1] + p[1]*m[5] + p[2]*m[9], p[0]*m[2] + p[1]*m[6] + p[2]*m[10]}
}
func (x *Effect) SetTransform(m render.M4) {
	x.Origin = render.V3{m[12], m[13], m[14]}
	m[12], m[13], m[14] = 0, 0, 0
	x.basis = nativeEffectBasis(m)
}

// SetMeteorTransform carries the CMeteor placement into the existing EFT.
// Mesh bodies simulate locally and follow the drop; billboard trails retain
// world-space births. This changes matrices only, not geometry or draw passes.
func (x *Effect) SetMeteorTransform(m render.M4) {
	x.SetTransform(m)
	x.meteor, x.meteorWorld = true, x.basis
	x.meteorWorld[12], x.meteorWorld[13], x.meteorWorld[14] = x.Origin[0], x.Origin[1], x.Origin[2]
}

// FinishMeteor hides the falling mesh immediately at terrain contact and
// stops new births; existing world-space trail particles drain normally.
func (x *Effect) FinishMeteor() {
	x.Stop()
	for i := range x.emitters {
		if x.emitters[i].geometry != nil {
			clear(x.emitters[i].particles)
			x.emitters[i].particles = x.emitters[i].particles[:0]
		}
	}
}
func (x *Effect) Stop() { x.stopped = true }
func (x *Effect) Done() bool {
	for _, e := range x.emitters {
		if len(e.particles) > 0 {
			return false
		}
		if !x.stopped && (e.definition.Loop != 0 || x.Time < e.starts || e.time <= e.definition.Parameters().Duration) {
			return false
		}
	}
	return true
}

// nativeEffectBasis follows Game.exe 0x5426e0: reflect the input forward
// in X/Z, rebuild the orthogonal axes, then store the native transposed
// basis with its Z signs. EFT placements do not use the object Basis layout.
func nativeEffectBasis(world render.M4) render.M4 {
	f := render.V3{-world[8], world[9], -world[10]}.Norm()
	up := render.V3{world[4], world[5], world[6]}
	r := up.Cross(f).Norm()
	u := f.Cross(r)
	return render.M4{r[0], u[0], -f[0], 0, r[1], u[1], -f[1], 0, -r[2], u[2], f[2], 0, 0, 0, 0, 1}
}

// MON attachments pass bone * creature world directly (native 0x431d29).
// Existing particles stay in world space; only births and zero-life anchors
// follow later pose updates. Skill/world placement basis conversion is separate.
func (x *Effect) SetBoneTransform(m render.M4) {
	x.Origin = render.V3{m[12], m[13], m[14]}
	m[12], m[13], m[14] = 0, 0, 0
	x.basis = m
}

// A fixed single-axis tilt needs just the two nontrivial matrix coefficients.
// Derive them from the rounded half-angle quaternion used by effectRotation.
type effectTilt struct{ cosine, sine float32 }

func newEffectTilt(axis int32, angle float32) effectTilt {
	if angle == 0 {
		return effectTilt{}
	}
	if axis < 1 || axis > 3 {
		return effectTilt{cosine: 1}
	}
	half := float64(angle) * .5
	sin, cos := float32(math.Sin(half)), float32(math.Cos(half))
	return effectTilt{1 - 2*(sin*sin), 2 * (sin * cos)}
}

func (t effectTilt) matrix(axis int32) render.M4 {
	m := render.Identity()
	c, s := t.cosine, t.sine
	switch axis {
	case 1:
		m[5], m[10], m[6], m[9] = c, c, s, -s
	case 2:
		m[0], m[10], m[8], m[2] = c, c, s, -s
	case 3:
		m[0], m[5], m[1], m[4] = c, c, s, -s
	}
	return m
}

// The original mesh path (0x55b1f0) first builds tilt * placement, then
// prepends axial spin. Reversing them makes a tilted projectile precess.
func particleRotation(d *effect.Emitter, fixedTilt effectTilt, angle float32, base render.M4, mesh bool) render.M4 {
	if fixedTilt == (effectTilt{}) {
		if angle == 0 {
			return base
		}
		return effectRotation(d.RotationAxis, angle).MulAffine(base)
	}
	tilt := fixedTilt.matrix(d.Unk30)
	if angle == 0 {
		return tilt.MulAffine(base)
	}
	spin := effectRotation(d.RotationAxis, angle)
	if mesh {
		return spin.MulAffine(tilt).MulAffine(base)
	}
	return tilt.MulAffine(spin).MulAffine(base)
}
