//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestHelpPageGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "help", Width: 1024, Height: 768, Hidden: true, Debug: true})
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
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	a.gameMenu.Open(ui.MenuHelp)
	for _, step := range []struct {
		name       string
		tab, entry int
	}{{"help-move", 0, 0}, {"help-interface", 2, 1}, {"help-bless", 3, 0}} {
		a.gameMenu.HelpSelect(step.tab, step.entry)
		gl.Viewport(0, 0, 1024, 768)
		gl.ClearColor(.1, .1, .1, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(1024, 768)
		a.gameMenu.Draw(a.sprites, a.font, 1024, 768, a.white)
		a.sprites.End()
		if err := render.SavePNG(filepath.Join(dir, step.name+".png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
}
