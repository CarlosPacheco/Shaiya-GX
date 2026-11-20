//go:build integration

package app

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestEvolutionModelAndClickBarGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires OpenGL and the external game assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	const width, height = 800, 600
	window, err := platform.Open(platform.Config{Width: width, Height: height, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err = render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state, a.wv = game.New(), &worldView{}
	a.state.SelfID = 1
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Level: 60, Name: "Evolution"}
	a.refreshSelf(a.wv)
	for i, id := range []uint16{2049, 413, 679} {
		a.state.PlayerForms = map[uint32]proto.PlayerFormAppearance{1: {
			User: 1, Kind: proto.FormEvolution, Skill: 127, Rank: uint8(i + 1),
		}}
		a.prepareTransforms()
		actor := a.wv.transforms[1]
		if actor == nil || actor.actor == nil || actor.typeID != id || !a.wv.self.Hidden {
			t.Fatalf("rank %d failed to load installed model %d", i+1, id)
		}
	}
	a.state.Evolution = proto.EvolutionBar{Generation: 1, Actions: [2]proto.EvolutionAction{{Skill: 119}}}
	a.confirmEvolutionBar()
	gl.Viewport(0, 0, width, height)
	gl.ClearColor(.12, .2, .28, 1)
	pixels := func(draw func()) []byte {
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(width, height)
		draw()
		a.sprites.End()
		out := make([]byte, width*height*4)
		gl.ReadPixels(0, 0, width, height, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&out[0]))
		return out
	}
	actual := pixels(func() { a.drawEvolutionBar(width, height) })
	if bytes.Equal(actual, pixels(func() {})) {
		t.Fatal("the Evolution bar did not draw")
	}
	x, y := a.evolutionOrigin(width, height)
	cells := pixels(func() {
		ui.DrawQuickPotions(a.sprites, a.font, a.assets.Texture("interface/QuickPotion.png"), a.white, x, y, [2]ui.ActionSlot{})
	})
	if bytes.Equal(actual, cells) {
		t.Fatal("the temporary skill and unlabeled cells did not replace X/Z")
	}
	pixels(func() {
		a.drawQuickPotions(width, height)
		a.drawEvolutionBar(width, height)
	})
	if out := os.Getenv("SHAIYA_TEST_ARTIFACTS"); out != "" {
		if err := render.SavePNG(filepath.Join(out, "evolution-bar.png"), width, height); err != nil {
			t.Fatal(err)
		}
	}
	delete(a.state.PlayerForms, 1)
	a.state.Evolution = proto.EvolutionBar{}
	a.prepareTransforms()
	if a.wv.self.Hidden || len(a.wv.transforms) != 0 {
		t.Fatal("the old creature did not disappear at the end of Evolution")
	}
}
