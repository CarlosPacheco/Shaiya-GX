//go:build integration

package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"os"
	"path"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestOriginalCollisionMapsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original data")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	for _, id := range []uint16{1, 2, 3} {
		t.Run(string(rune('0'+id)), func(t *testing.T) {
			started := time.Now()
			w, err := LoadMap(NewAssets(arc), id)
			if err != nil {
				t.Fatal(err)
			}
			defer w.Close()
			if w.collision == nil || w.collision.Len() == 0 {
				t.Fatal("map has no collision geometry")
			}
			t.Logf("map %d: %d static triangles, %d moving; load %s", id, w.collision.Len(), len(w.movingColliders), time.Since(started))
		})
	}
}

// CPU-only audit of all shipped numeric WLDs. No texture residency or graphics
// setting may decide which maps/assets have collision geometry.
func TestOriginalCollisionContentAudit(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_COLLISION_AUDIT") != "1" {
		t.Skip("requires the complete local original data")
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	maps := []string{}
	arc.Walk(func(file string, _ *archive.File) {
		lower := strings.ToLower(file)
		if path.Dir(lower) != "world" || path.Ext(lower) != ".wld" {
			return
		}
		if _, err := strconv.ParseUint(strings.TrimSuffix(path.Base(file), path.Ext(file)), 10, 16); err == nil {
			maps = append(maps, file)
		}
	})
	sort.Strings(maps)
	for _, file := range maps {
		raw, err := arc.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		layout, err := world.ParseWLD(raw)
		if err != nil {
			t.Errorf("%s: %v", file, err)
			continue
		}
		w := &World{WLD: layout}
		if err := w.prepareCollisions(NewAssets(arc)); err != nil {
			t.Errorf("%s: %v", file, err)
			continue
		}
		if len(layout.Objects.Instances) > 0 {
			t.Logf("%s: special Objects assets=%v", file, layout.Objects.Assets)
		}
		runtime.GC()
	}
	t.Logf("Audited %d original numeric maps without GL", len(maps))
}

func BenchmarkPlayerCollisionMarket(b *testing.B) {
	directory := os.Getenv("SHAIYA_DATA")
	if directory == "" {
		b.Skip("local assets required")
	}
	arc, err := archive.OpenDirectory(directory)
	if err != nil {
		b.Fatal(err)
	}
	defer arc.Close()
	raw, err := arc.ReadFile("world/1.wld")
	if err != nil {
		b.Fatal(err)
	}
	layout, err := world.ParseWLD(raw)
	if err != nil {
		b.Fatal(err)
	}
	w := &World{WLD: layout}
	if err := w.prepareCollisions(NewAssets(arc)); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.MovePlayer(render.V3{995.8, 13, 797.3}, render.V3{.08, 0, .08})
	}
}
