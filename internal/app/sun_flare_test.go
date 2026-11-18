package app

import (
	"math"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

func TestSunFlareProjectionAndFade(t *testing.T) {
	eye := render.V3{0, 10, 0}
	view := render.LookAtLH(eye, eye.Add(sunPosition), render.V3{0, 1, 0})
	projection := render.PerspectiveFovLH(math.Pi/4, 1.25, 10, 100000)
	x, y, visible := projectSun(view, projection)
	if !visible || math.Abs(float64(x))+math.Abs(float64(y)) > .001 {
		t.Fatalf("sun should project near the centre: %v %v %v", x, y, visible)
	}
	back := render.LookAtLH(eye, eye.Sub(sunPosition), render.V3{0, 1, 0})
	if _, _, visible := projectSun(back, projection); visible {
		t.Fatal("sun visible behind the camera")
	}
	field := &scene.World{Sky: &scene.Sky{}, WLD: &world.WLD{}}
	var flare sunFlare
	flare.update(field, eye, view, projection, 400, .125)
	if flare.strength != .5 {
		t.Fatalf("halfway through the fade: %v", flare.strength)
	}
	flare.update(field, eye, back, projection, 400, .125)
	if flare.strength != 0 {
		t.Fatal("offscreen flare")
	}
	field.WLD.Dungeon = true
	flare.update(field, eye, view, projection, 400, 1)
	if flare.strength != 0 || flare.visibility != 0 {
		t.Fatal("dungeon carried outdoor flare")
	}
}
