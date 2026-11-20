package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
	"testing"
)

func TestMeteorBodyFollowsControllerWhileTrailKeepsWorldBirth(t *testing.T) {
	mesh := &effect.Emitter{Mesh: 0, Position: [3]float32{1, 0, 0}, InitialDelay: 1, Unk13: 1, Unk12: 60, Unk14: 60, Unk16: 1}
	trail := *mesh
	trail.Mesh = -1
	x := &Effect{emitters: []effectEmitter{{definition: mesh, geometry: &effectGeometry{source: &effect.Mesh3DE{}}}, {definition: &trail}}}
	from := render.Basis(render.V3{10, 15, 20}, render.V3{-1, 0, 0}, render.V3{0, 1, 0})
	x.SetMeteorTransform(from)
	x.Update(1.0 / 60)
	if len(x.emitters[0].particles) != 1 || len(x.emitters[1].particles) != 1 {
		t.Fatal("missing ordinary EFT births")
	}
	body := x.emitters[0].particles[0]
	smoke := x.emitters[1].particles[0]
	before := x.particlePosition(smoke, &trail)
	to := render.Basis(render.V3{10, 4, 27.5}, render.V3{0, 0, -1}, render.V3{0, 1, 0})
	x.SetMeteorTransform(to)
	if got, want := x.particlePosition(body, mesh), (render.V3{11, 4, 27.5}); got != want {
		t.Fatal("mesh stayed at birth instead of falling", got, want)
	}
	if got := x.particlePosition(smoke, &trail); got != before {
		t.Fatal("trail moved with body", got, before)
	}
	if len(x.emitters) != 2 || len(x.emitters[0].particles) != 1 || len(x.emitters[1].particles) != 1 {
		t.Fatal("placement created extra emitters/particles")
	}
}

func TestMeteorAxialSpinPreservesAuthoredDiagonal(t *testing.T) {
	d := &effect.Emitter{Unk30: 3, RotationAxis: 2}
	placement := render.Basis(render.V3{}, render.V3{-.6, 0, -.8}, render.V3{0, 1, 0})
	tilt := float32(333 * math.Pi / 180)
	fixedTilt := newEffectTilt(d.Unk30, tilt)
	first := effectPoint(render.V3{0, 1, 0}, particleRotation(d, fixedTilt, 0, placement, true))
	for frame := 0; frame < 120; frame++ {
		angle := float32(frame) * 10 / 60
		axis := effectPoint(render.V3{0, 1, 0}, particleRotation(d, fixedTilt, angle, placement, true))
		if axis.Sub(first).Len() > 1e-5 {
			t.Fatal("axial spin changed projectile direction", frame, axis, first)
		}
	}
	// The old composition turns a fixed tilted axis into a moving cone.
	old := effectPoint(render.V3{0, 1, 0}, effectRotation(3, tilt).Mul(effectRotation(2, math.Pi/2)).Mul(placement))
	if old.Sub(first).Len() < .1 {
		t.Fatal("fixture does not reproduce the former wobble")
	}
}

func TestMeteorTiltFacesAlongDiagonalTravelInNativeEFTSpace(t *testing.T) {
	d := &effect.Emitter{Unk30: 3, RotationAxis: 2}
	fixedTilt := newEffectTilt(d.Unk30, float32(333*math.Pi/180))
	for _, dir := range []render.V3{{1, 0, 0}, {0, 0, 1}, {-1, 0, 0}, {0, 0, -1}, {.6, 0, .8}} {
		var x Effect
		x.SetMeteorTransform(render.Basis(render.V3{}, dir.Scale(-1), render.V3{0, 1, 0}))
		velocity := render.V3{-dir[2] * 7.5, -15, dir[0] * 7.5}.Norm()
		for frame := 0; frame < 60; frame++ {
			axis := effectPoint(render.V3{0, 1, 0}, particleRotation(d, fixedTilt, float32(frame)/6, x.meteorWorld, true)).Norm()
			if axis.Dot(velocity.Scale(-1)) < .999 {
				t.Fatal("mesh tilt opposes actual fall", dir, frame, axis, velocity)
			}
		}
	}
}

func TestMeteorContactRemovesBodyAndStopsBirthsButDrainsTrail(t *testing.T) {
	mesh := &effect.Emitter{Mesh: 0, Loop: 1, InitialDelay: 1, Unk13: 1, Unk12: 60, Unk14: 60, Unk16: 1}
	trail := *mesh
	trail.Mesh = -1
	x := &Effect{emitters: []effectEmitter{{definition: mesh, geometry: &effectGeometry{source: &effect.Mesh3DE{}}}, {definition: &trail}}}
	x.SetMeteorTransform(render.Identity())
	x.Update(1.0 / 60)
	x.FinishMeteor()
	if len(x.emitters[0].particles) != 0 || len(x.emitters[1].particles) != 1 {
		t.Fatal("contact left body alive or erased trail")
	}
	x.Update(.25)
	if len(x.emitters[0].particles) != 0 || len(x.emitters[1].particles) != 1 {
		t.Fatal("new births after contact")
	}
	for range 5 {
		x.Update(.25)
	}
	if !x.Done() {
		t.Fatal("finished projectile did not drain")
	}
}
