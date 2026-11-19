//go:build integration

package scene

import (
	"os"
	"runtime"
	"slices"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestDungeonEnvironmentFromWLDGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" || os.Getenv("SHAIYA_TEST_DUNGEON_ENVIRONMENT") != "1" {
		t.Skip("requires prepared GX dungeon WLDs and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window, err := platform.Open(platform.Config{Width: 160, Height: 120, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	assets := NewAssets(arc)
	scene, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer scene.Delete()
	eye := render.V3{0, 100, 0}
	scene.SetCameraView(eye, eye.Add(render.V3{0, .65, 1}), render.V3{0, 1, 0})
	scene.Proj = render.PerspectiveFovLH(1, 4.0/3, .1, 400)
	scene.SkyProj = render.PerspectiveFovLH(1, 4.0/3, 10, 100000)
	gl.Viewport(0, 0, 160, 120)
	pixels := func(s *Sky) []byte {
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		scene.Begin()
		s.Draw(scene, eye)
		scene.End()
		frame := make([]byte, 160*120*4)
		gl.ReadPixels(0, 0, 160, 120, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&frame[0]))
		return frame
	}
	black := pixels(nil)
	skies := make(map[string][]byte)
	for _, tc := range []struct {
		id, sky          string
		fogStart, fogEnd float32
	}{
		{"3", "", 0, 0},
		{"39", "Sky_A1.bmp", 0, 0},
		{"40", "sky_B1.bmp", 0, 0},
		{"59", "", 200, 300},
		{"61", "", 200, 300},
	} {
		w, err := LoadDungeon(assets, "world/"+tc.id+".wld")
		if err != nil {
			t.Fatalf("map %s: %v", tc.id, err)
		}
		defer w.Close()
		if w.FogStart != tc.fogStart || w.FogEnd != tc.fogEnd {
			t.Fatalf("map %s: fog = %v..%v", tc.id, w.FogStart, w.FogEnd)
		}
		frame := pixels(w.Sky)
		if tc.sky == "" {
			if w.Sky != nil || !slices.Equal(frame, black) {
				t.Fatalf("map %s: dungeon without sky lost its black background", tc.id)
			}
			continue
		}
		if w.WLD.Sky != tc.sky || w.Sky == nil || w.Sky.DomeTex == nil ||
			w.Sky.Clouds[0] == nil || w.Sky.Clouds[1] == nil || slices.Equal(frame, black) {
			t.Fatalf("map %s: authored sky or clouds did not load and render", tc.id)
		}
		clouds := w.Sky.Clouds
		w.Sky.Clouds = [2]*render.Texture{}
		withoutClouds := pixels(w.Sky)
		w.Sky.Clouds = clouds
		if slices.Equal(frame, withoutClouds) {
			t.Fatalf("map %s: cloud layers did not contribute to the image", tc.id)
		}
		skies[tc.id] = frame
	}
	if slices.Equal(skies["39"], skies["40"]) {
		t.Fatal("the two arena skies rendered identically")
	}
}
