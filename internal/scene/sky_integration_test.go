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
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

func TestOvercastSkySurvivesFieldStreaming(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window, err := platform.Open(platform.Config{Title: "streamed storm sky", Width: 128, Height: 96, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err := render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	w, err := loadStreamedField(NewAssets(arc), "1")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if w.Sky == nil || w.Sky.Overcast == nil || w.Sky.Clouds[0] == nil {
		t.Fatal("map 1 sky assets are missing")
	}
	w.stream.trim()
	if !gl.IsTexture(w.Sky.Overcast.ID) {
		t.Fatal("streaming released the live overcast texture")
	}

	s, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	eye := render.V3{911, 55, 1252}
	s.SetCameraView(eye, eye.Add(render.V3{0, .65, 1}), render.V3{0, 1, 0})
	s.Proj = render.PerspectiveFovLH(1, 4.0/3, .1, 200)
	s.SkyProj = render.PerspectiveFovLH(1, 4.0/3, 10, 100000)
	gl.Viewport(0, 0, 128, 96)
	pixels := func(blend float32) []byte {
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		s.Begin()
		w.Sky.DrawWeather(s, eye, blend)
		s.End()
		frame := make([]byte, 128*96*4)
		gl.ReadPixels(0, 0, 128, 96, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&frame[0]))
		return frame
	}
	clear, storm := pixels(0), pixels(1)
	center := (48*128 + 64) * 4
	if slices.Equal(clear, storm) || storm[center]|storm[center+1]|storm[center+2] == 0 || !slices.Equal(pixels(0), clear) {
		t.Fatalf("sky transition did not render and restore: clear %v storm %v", clear[center:center+4], storm[center:center+4])
	}
	clouds := w.Sky.Clouds
	w.Sky.Clouds = [2]*render.Texture{}
	withoutClouds := pixels(1)
	w.Sky.Clouds = clouds
	if slices.Equal(storm, withoutClouds) {
		t.Fatal("clouds did not contribute to the storm sky")
	}
	t.Logf("map 1 after streaming: clear %v storm %v", clear[center:center+4], storm[center:center+4])
}
