//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeTargetEffectsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 1024, Height: 768, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
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
	if err = a.Preview("world-inventory"); err != nil {
		t.Fatal(err)
	}
	a.inventory.open = false
	a.clock = 100.5
	for id, e := range a.state.Entities {
		if e.Kind == game.KindMob {
			a.state.Target = id
			break
		}
	}
	a.targetEffects.requestedID = a.state.Target
	a.loadSkills()
	positive := uint16(65535)
	for key, d := range a.skillDefs {
		if key.Level == 1 && d.Effect == 4 {
			positive = min(positive, key.ID)
		}
	}
	for _, id := range []uint16{33, positive, 33, positive, 33, positive, 33, positive, 33} {
		d, ok := a.skillDefinition(id, 1)
		if !ok {
			t.Fatal("missing original icon", id)
		}
		a.targetEffects.rows = append(a.targetEffects.rows, buffBadge{buff: proto.ActiveBuff{Skill: id, Level: 1}, def: d, until: a.clock + 20, duration: 30, timed: true})
	}
	x, y := a.targetEffectSlot(1024, 768, 8)
	if x != 296 || y != 80 {
		t.Fatal("wrong native icon row pitch", x, y)
	}
	a.lastMouseX, a.lastMouseY = x+4, y+4
	gl.Viewport(0, 0, 1024, 768)
	a.Draw(1024, 768)
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir != "" {
		if err = render.SavePNG(filepath.Join(dir, "target-effects.png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatal("target effects GL error", code)
	}
}
