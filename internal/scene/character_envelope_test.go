package scene

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func envelopeFixture() *Character {
	clip := poseTestClip(8)
	for i := range clip.ani.Bones {
		clip.ani.Bones[i].Rot = []model.RotKey{{Frame: 0, Q: [4]float32{0, 0, 0, 1}}, {Frame: 30, Q: [4]float32{0, .8, 0, .6}}}
	}
	clip.reach = clip.makeReach()
	part := &skinPart{bounded: true}
	for i := range 8 {
		bind := render.Identity()
		bind[13] = -float32(i) * .2
		part.bones = append(part.bones, bind)
		part.bounds = append(part.bounds, boneBounds{used: true, min: render.V3{-.4, 0, -.2}, max: render.V3{.4, 2, .2}})
	}
	return &Character{clip: clip, parts: []*skinPart{part}, World: render.Identity(), attachments: []attachment{{part: &Part{bounded: true, boundMin: render.V3{-1, -1, -1}, boundMax: render.V3{3, 1, 1}}, bone: 7, extra: render.Identity()}}}
}

func assertEnvelopeContainsExact(t *testing.T, c *Character) {
	t.Helper()
	lo, hi, ok := c.coarseBodyBounds()
	if !ok {
		t.Fatal("fixture envelope invalid")
	}
	exactLo, exactHi, ok := c.WorldBounds()
	if !ok {
		t.Fatal("fixture exact bounds invalid")
	}
	for axis := range 3 {
		if lo[axis] > exactLo[axis] || hi[axis] < exactHi[axis] {
			t.Fatalf("axis %d envelope %v..%v excludes exact %v..%v", axis, lo, hi, exactLo, exactHi)
		}
	}
}

func TestCharacterEnvelopeContainsAnimatedBounds(t *testing.T) {
	c := envelopeFixture()
	rng := rand.New(rand.NewPCG(3, 7))
	for range 500 {
		c.Time = rng.Float32()*5 - .1
		// General affine worlds, including nonuniform scale, reflection and shear.
		for i := range 12 {
			if i%4 != 3 {
				c.World[i] = rng.Float32()*8 - 4
			}
		}
		c.World[12], c.World[13], c.World[14] = rng.Float32()*1000, rng.Float32()*1000, rng.Float32()*1000
		assertEnvelopeContainsExact(t, c)
	}
	c.clip = poseTestClip(1)
	c.clip.reach = c.clip.makeReach()
	assertEnvelopeContainsExact(t, c) // Extra skin bones use the identity fallback.
}

func TestCharacterEnvelopeSkipsPoseButPreservesOtherCameras(t *testing.T) {
	c := envelopeFixture()
	c.World[12] = 1000
	c.cloak = &cloakInstance{}
	f := render.ViewFrustum(render.Identity(), render.PerspectiveFovLH(math.Pi/4, 1, .1, 100))
	if c.Visible(f, [4]float32{}) || c.poseReady || c.lastFinals != nil {
		t.Fatal("offscreen rejection sampled pose")
	}
	// A different camera can see the same actor immediately, at unchanged time.
	view := render.LookAtLH(render.V3{1000, 2, -10}, render.V3{1000, 2, 0}, render.V3{0, 1, 0})
	if !c.Visible(render.ViewFrustum(view, render.PerspectiveFovLH(math.Pi/4, 1, .1, 100)), [4]float32{}) || !c.poseReady {
		t.Fatal("other pass reused first camera rejection")
	}
	c.poseReady = false
	c.lastFinals = nil
	c.boundReady = false
	if c.Visible(f, [4]float32{-1, 0, 0, 0}) || c.poseReady {
		t.Fatal("clip-plane rejection sampled pose")
	}
}

func TestCharacterEnvelopeInvalidationAndFallback(t *testing.T) {
	c := envelopeFixture()
	assertEnvelopeContainsExact(t, c)
	c.attachments[0].extra[12] = 100
	c.boundReady = false
	assertEnvelopeContainsExact(t, c)
	c.Mounted = true
	c.boundReady = false
	assertEnvelopeContainsExact(t, c)
	c.parts = []*skinPart{{bounded: true, bounds: []boneBounds{{used: true, min: render.V3{-20, -1, -1}, max: render.V3{20, 1, 1}}}}}
	c.boundReady = false
	assertEnvelopeContainsExact(t, c)
	c.cloak = &cloakInstance{}
	assertEnvelopeContainsExact(t, c)
	c.cloak = nil
	c.World[3] = 1
	if _, _, ok := c.coarseBodyBounds(); ok {
		t.Fatal("projective world accepted")
	}
	c.World = render.Identity()
	c.parts[0].bounded = false
	c.envelope.ready = false
	if _, _, ok := c.coarseBodyBounds(); ok {
		t.Fatal("unknown geometry accepted")
	}
	c = envelopeFixture()
	c.attachments[0].part.bounded = false
	c.envelope.ready = false
	if _, _, ok := c.coarseBodyBounds(); ok {
		t.Fatal("unknown equipment accepted")
	}
	c = envelopeFixture()
	c.World[12] = float32(math.NaN())
	if _, _, ok := c.coarseBodyBounds(); ok {
		t.Fatal("nonfinite world accepted")
	}
}

func TestClipReachRejectsUncertainAnimation(t *testing.T) {
	for _, alter := range []func(*Clip){
		func(c *Clip) { c.ani.Bones[0].Rot = []model.RotKey{{Q: [4]float32{0, 0, 0, 2}}} },
		func(c *Clip) { c.ani.Bones[0].Trans[0].T[0] = float32(math.NaN()) },
		func(c *Clip) { c.ani.Bones[0].Trans[1].Frame = 0 },
		func(c *Clip) {
			c.ani.Bones[0].Rot = []model.RotKey{{Frame: 1, Q: [4]float32{0, 0, 0, 1}}, {Frame: 0, Q: [4]float32{0, 0, 0, 1}}}
		},
	} {
		c := poseTestClip(2)
		alter(c)
		if c.makeReach() != nil {
			t.Fatal("uncertain clip accepted")
		}
	}
}

func TestPreculledNameAnchorSamplesCurrentCrown(t *testing.T) {
	c := envelopeFixture()
	c.head = c.parts[0]
	c.head.min, c.head.max = render.V3{-.4, 0, -.2}, render.V3{.4, 2, .2}
	c.head.top = render.SkinVertex{Weight: .3, Bone0: 1, Bone1: 7}
	c.preparePose()
	c.Time = .73
	got, ok := c.NameAnchor()
	if !ok {
		t.Fatal("preculled crown unavailable")
	}
	if c.poseTime == c.Time {
		t.Fatal("label rebuilt all part palettes")
	}
	c.preparePose()
	expected, ok := c.NameAnchor()
	if !ok || got != expected {
		t.Fatalf("stale label after precull: %v != %v", got, expected)
	}
}

func TestCharacterEnvelopeRejectsOverflowRisk(t *testing.T) {
	c := envelopeFixture()
	c.parts[0].bones[0][0] = 1e30
	if _, _, ok := c.coarseBodyBounds(); ok {
		t.Fatal("extreme inverse bind accepted")
	}
	c = envelopeFixture()
	c.World[0] = 1e30
	if _, _, ok := c.coarseBodyBounds(); ok {
		t.Fatal("extreme world accepted")
	}
}
