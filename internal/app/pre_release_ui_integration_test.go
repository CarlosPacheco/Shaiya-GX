//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestStatusAndPotionNativeFieldsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original data")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 800, Height: 600, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state = game.New()
	a.state.SelfID = 1
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Mode: 3}
	var slots [2]ui.ActionSlot
	for index := range slots {
		def := a.itemsItem(25, uint8(index+4))
		if def == nil {
			t.Fatal("missing potion definition")
		}
		file, uv, ok := ui.ItemIcon(25, def.Icon)
		if !ok {
			t.Fatal("missing potion icon")
		}
		slots[index] = ui.ActionSlot{Icon: a.assets.Texture("interface/" + file), UV: uv, Count: 255}
	}
	out := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if out == "" {
		out = t.TempDir()
	}
	for index, value := range []uint8{0, 6} {
		view := a.statusState()
		view.Element = elementName(value, a.msg)
		if view.Mode != "" {
			t.Fatal("mode must remain empty")
		}
		gl.Viewport(0, 0, 800, 600)
		gl.ClearColor(.15, .15, .15, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(800, 600)
		a.statusUI.Draw(a.sprites, a.font, 800, 600, view)
		ui.DrawQuickPotions(a.sprites, a.font, a.assets.Texture("interface/QuickPotion.png"), a.white, 80, 350, slots)
		a.sprites.End()
		if code := gl.GetError(); code != gl.NO_ERROR {
			t.Fatal(code)
		}
		if err := render.SavePNG(filepath.Join(out, []string{"status-neutral-potions.png", "status-element-potions.png"}[index]), 800, 600); err != nil {
			t.Fatal(err)
		}
	}
}
