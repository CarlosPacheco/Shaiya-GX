//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestPlatePartyIconGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "plate", Width: 400, Height: 200, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(true); err != nil {
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
	gl.Viewport(0, 0, 400, 200)
	gl.ClearColor(.2, .3, .2, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(400, 200)
	for i, name := range []string{"Friend", "Leader"} {
		y := 60 + i*60
		cx := 200
		left := cx - a.font.Width(name)/2
		a.font.DrawName(a.sprites, name, left, y, white)
		file := "interface/party_member.png"
		if i == 1 {
			file = "interface/party_jang.png"
		}
		if tex := a.assets.Texture(file); tex != nil {
			a.sprites.Draw(tex, float32(left-18), float32(y-3), 16, 16, 0, 0, 1, 1, white)
		}
	}
	a.sprites.End()
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if err := render.SavePNG(filepath.Join(dir, "plate-icon.png"), 400, 200); err != nil {
			t.Fatal(err)
		}
	}
}
