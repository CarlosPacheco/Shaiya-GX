package scene

import (
	"slices"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func poseTestClip(bones int) *Clip {
	c := &Clip{ani: &model.ANI{End: 30}}
	for i := range bones {
		rest := render.Identity()
		rest[12], rest[13] = float32(i)*.1, .3
		c.restLocal = append(c.restLocal, rest)
		c.ani.Bones = append(c.ani.Bones, model.AniBone{Parent: int32(i - 1), Trans: []model.TransKey{{Frame: 0, T: [3]float32{0, .3, 0}}, {Frame: 30, T: [3]float32{.1, .5, 0}}}})
	}
	return c
}

func TestPoseReusePreservesMatricesAndBounds(t *testing.T) {
	clip := poseTestClip(4)
	binds := []render.M4{render.Identity(), render.Identity(), render.Identity(), render.Identity()}
	different := slices.Clone(binds)
	different[1][12] = 2
	a := &skinPart{bones: binds, bounded: true, bounds: []boneBounds{{used: true, min: render.V3{-1, -1, -1}, max: render.V3{1, 1, 1}}}}
	b := &skinPart{bones: slices.Clone(binds), bounded: true, bounds: a.bounds}
	d := &skinPart{bones: different, bounded: true, bounds: a.bounds}
	actor := &Character{clip: clip, parts: []*skinPart{a, b, d}, World: render.Identity()}
	other := &Character{clip: clip, parts: []*skinPart{{bones: binds}}}
	actor.preparePose()
	storage := &actor.lastFinals[0]
	for _, time := range []float32{.1, .5, .99, 1.2, 0} {
		actor.Time = time
		actor.preparePose()
		if &actor.lastFinals[0] != storage {
			t.Fatal("actor finals reallocated")
		}
		if &a.pose[0] != &b.pose[0] {
			t.Fatal("identical binds did not share palette")
		}
		if &a.pose[0] == &d.pose[0] {
			t.Fatal("different binds shared palette")
		}
		expected := clip.Finals(time)
		for _, part := range actor.parts {
			for i, bind := range part.bones {
				if part.pose[i] != bind.Mul(expected[i]) {
					t.Fatalf("pose changed at t=%v bone=%d", time, i)
				}
			}
		}
		// Compare bounds with independently posed parts and the original box math.
		lo, hi := render.V3{}, render.V3{}
		has := false
		for _, part := range actor.parts {
			for i, box := range part.bounds {
				if box.used {
					unionTransformedBox(&lo, &hi, &has, box.min, box.max, part.bones[i].Mul(expected[i]).Mul(actor.World))
				}
			}
		}
		for axis := range 3 {
			lo[axis] -= .01
			hi[axis] += .01
		}
		gotLo, gotHi, ok := actor.WorldBounds()
		if !ok || gotLo != lo || gotHi != hi {
			t.Fatal("bounds changed")
		}
		other.Time = time + .2
		other.preparePose()
		if !slices.Equal(actor.lastFinals, expected) {
			t.Fatal("another actor overwrote pose")
		}
	}
	// A changed outfit must rebuild sharing even at an unchanged animation time.
	actor.parts = []*skinPart{d, b}
	actor.preparePose()
	if &d.pose[0] == &b.pose[0] {
		t.Fatal("outfit replacement reused incompatible palette")
	}
	// Clips may change bone counts, including growing beyond existing capacity.
	actor.clip = poseTestClip(8)
	if len(actor.preparePose()) != 8 {
		t.Fatal("clip growth lost bones")
	}
	actor.clip = poseTestClip(1)
	if len(actor.preparePose()) != 1 || b.pose[3] != binds[3] {
		t.Fatal("short clip did not preserve identity fallback")
	}
}

func TestClipCachedRestMatchesSampling(t *testing.T) {
	clip := poseTestClip(4)
	// Include a bone with no keys, which must use the decomposed rest transform.
	clip.ani.Bones[1].Trans = nil
	for _, rest := range clip.restLocal {
		q, t := render.MatrixQuat(rest)
		clip.restRotation = append(clip.restRotation, q)
		clip.restTranslation = append(clip.restTranslation, t)
	}
	reference := *clip
	reference.restRotation, reference.restTranslation = nil, nil
	for _, time := range []float32{0, .1, .5, 1, 2.3} {
		if !slices.Equal(clip.Finals(time), reference.Finals(time)) {
			t.Fatalf("cached rest changed pose at %v", time)
		}
	}
	actor := &Character{clip: clip, parts: []*skinPart{{bones: []render.M4{render.Identity()}}}}
	actor.preparePose()
	if allocs := testing.AllocsPerRun(100, func() { actor.Time += .001; actor.preparePose() }); allocs != 0 {
		t.Fatalf("steady pose allocs=%v", allocs)
	}
}

func BenchmarkCharacterPreparePose(b *testing.B) {
	clip := poseTestClip(64)
	binds := make([]render.M4, 64)
	for i := range binds {
		binds[i] = render.Identity()
	}
	actor := &Character{clip: clip}
	for range 6 {
		actor.parts = append(actor.parts, &skinPart{bones: slices.Clone(binds)})
	}
	actor.preparePose()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		actor.Time += .001
		if actor.Time > 1 {
			actor.Time = 0
		}
		actor.preparePose()
	}
}
