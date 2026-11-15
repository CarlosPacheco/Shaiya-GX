//go:build integration

package app

import (
	"bytes"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeSpeedStatsAndTooltipsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 1024, Height: 768, Hidden: true})
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
	if err := a.Preview("world-stream"); err != nil {
		t.Fatal(err)
	}
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	setSpeed := func(attack, move uint8) {
		w := proto.NewWriter(proto.CharacterDeathState)
		w.U32(a.state.SelfID)
		w.U8(attack)
		w.U8(move)
		if _, err := a.state.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
	capture := func(name string, typ, id uint8, gem uint8) []byte {
		gl.Viewport(0, 0, 1024, 768)
		gl.ClearColor(.1, .13, .15, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(1024, 768)
		a.statusUI.Draw(a.sprites, a.font, 1024, 768, a.statusState())
		def := a.itemsItem(typ, id)
		if def == nil {
			t.Fatal("missing native definition", typ, id)
		}
		item := proto.InventoryItem{Type: typ, TypeID: id, Count: 1, Gems: [6]uint8{gem}}
		a.drawItemDetails(def, &item, 1, 0, 0, 1024, 768, "")
		pos := [2]int{50, 50}
		a.drawSharedTooltipAt(1024, 768, &pos)
		a.sprites.End()
		pixels := make([]byte, 1024*768*4)
		gl.ReadPixels(0, 0, 1024, 768, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		if err := render.SavePNG(filepath.Join(dir, name), 1024, 768); err != nil {
			t.Fatal(err)
		}
		return pixels
	}
	setSpeed(5, 2)
	before := capture("stats-normal-sonic-amulet.png", 23, 57, 0)
	if s := a.statusState(); s.AttackSpeed != a.msg.Get(5276) || s.MoveSpeed != a.msg.Get(5276) {
		t.Fatal("normal native labels")
	}
	setSpeed(9, 3)
	after := capture("stats-fast-flash-ring.png", 22, 56, 0)
	// Both native input rectangles must visibly change after confirmed speed changes.
	for _, dy := range []int{345, 365} {
		changed := false
		for y := 114 + dy; y < 114+dy+16; y++ {
			i := ((768-y-1)*1024 + (1024 - 277 + 92)) * 4
			changed = changed || !bytes.Equal(before[i:i+95*4], after[i:i+95*4])
		}
		if !changed {
			t.Fatal("speed indicator produced no pixels", dy)
		}
	}
	capture("tooltip-flash-linked-weapon.png", 1, 1, 64)
	capture("tooltip-sonic-linked-boots.png", 21, 53, 106)
	if err := gl.GetError(); err != gl.NO_ERROR {
		t.Fatalf("speed UI GL error %#x", err)
	}
}
