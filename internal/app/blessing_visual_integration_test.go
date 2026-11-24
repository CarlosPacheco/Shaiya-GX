//go:build integration

package app

import (
	"bytes"
	"fmt"
	"math"
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
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

func TestBlessingBarUsesOriginalFooterGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "blessing", Width: 1024, Height: 768, Hidden: true, Debug: true})
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
	draw := func(width int, known bool, count, remaining uint32, reference bool, clock float64) []byte {
		gl.Viewport(0, 0, int32(width), 768)
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(width, 768)
		if !reference {
			a.bottom.Draw(a.sprites, a.font, a.white, width, 768, ui.HUDState{BlessKnown: known, BlessCount: count, BlessMax: 12288, BlessRemainingMS: remaining, Clock: clock}, false, false, false, false)
		} else {
			suffix, textX, x := "", 222, (width-1024)/2
			if width < 1024 {
				suffix, textX, x = "8", 170, (width-800)/2
			}
			a.sprites.DrawImage(a.assets.Texture("interface/main_bottom"+suffix+".png"), float32(x), 704)
			a.font.Draw(a.sprites, "0.0%", x+textX, 753, white, 70)
		}
		if reference && known {
			// Independent native draw constants: right anchored, 7 pixels,
			// original atlas U crop and V .5.. .9; active width uses time.
			x, y, total, name := 572, 753, 428, "main_bottom_bar.png"
			if width < 1024 {
				x, y, total, name = 446, 757, 336, "main_bottom_bar8.png"
			} else {
				x += (width - 1024) / 2
			}
			value := float32(count) / 12288
			if remaining > 0 {
				value = float32(remaining) / 600000
			}
			value = min(value, 1)
			filled := int(float32(total) * value)
			tex := a.assets.Texture("interface/" + name)
			a.sprites.Draw(tex, float32(x+total-filled), float32(y), float32(filled), 7, float32(total-filled)/512, .5, float32(total)/512, .9, white)
			if remaining > 0 && clock == .25 {
				v1 := float32(.9)
				if width < 1024 {
					v1 = 1
				}
				a.sprites.Flush()
				gl.BlendFunc(gl.SRC_ALPHA, gl.ONE)
				a.sprites.Draw(tex, float32(x+total-filled), float32(y), float32(filled), 7, float32(total-filled)/512, .5, float32(total)/512, v1, [4]float32{1, 1, 1, .5})
				a.sprites.Flush()
				gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
			}
		}
		if reference {
			x, y, pitch, size, suffix := (width-1024)/2+646, 714, 35, 64, ""
			if width < 1024 {
				x, y, pitch, size, suffix = (width-800)/2+505, 726, 27, 32, "8"
			}
			tex := a.assets.Texture("interface/main_bottom_button" + suffix + ".png")
			for i := 0; i < 8; i++ {
				a.sprites.Draw(tex, float32(x+i*pitch), float32(y), float32(size), 32, float32(i)/8, 0, float32(i+1)/8, .25, white)
			}
		}
		a.sprites.End()
		pixels := make([]byte, width*768*4)
		gl.ReadPixels(0, 0, int32(width), 768, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		return pixels
	}
	for _, width := range []int{800, 1024} {
		for _, tc := range [][2]uint32{{0, 0}, {149, 0}, {6144, 0}, {12288, 0}, {12288, 300000}} {
			actual := draw(width, true, tc[0], tc[1], false, 0)
			expected := draw(width, true, tc[0], tc[1], true, 0)
			if tc[1] > 0 {
				actual = draw(width, true, tc[0], tc[1], false, .25)
				expected = draw(width, true, tc[0], tc[1], true, .25)
			}
			if !bytes.Equal(actual, expected) {
				t.Fatalf("native blessing pixels differ width=%d count=%d remaining=%d", width, tc[0], tc[1])
			}
		}
	}
	// Real localized tooltip: current cumulative benefits, hover-only.
	a.state = game.New()
	a.state.Blessing.Known = true
	a.state.Blessing.Country = 0
	a.state.Blessing.Count = 1200
	a.state.Blessing.Maximum = 12288
	a.state.Blessing.Bonuses = [27]uint16{0: 6, 1: 6, 2: 6, 9: 5, 10: 6, 11: 6, 23: 1}
	a.lastMouseX, a.lastMouseY = 750, 755
	draw(1024, true, 1200, 0, false, 0)
	a.sprites.Begin(1024, 768)
	a.drawBlessingTooltip(1024, 768)
	a.sprites.End()
	// The original wide panel starts at x=624 here; the former 200px
	// panel started at 740. Check its empty left padding on the real GPU.
	var widePixel [4]byte
	gl.ReadPixels(626, 768-683, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&widePixel[0]))
	if widePixel[0] == 0 && widePixel[1] == 0 && widePixel[2] == 0 {
		t.Fatal("blessing tooltip still uses the narrow panel")
	}
	if lines := a.blessingLines(); len(lines) != 7 {
		t.Fatalf("real tooltip effects: %v", lines)
	}

	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if err := render.SavePNG(filepath.Join(dir, "blessing-bar.png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
	a.clock = 1
	a.addScreenNotice(8, "Destroyed altar")
	gl.ClearColor(0, 0, 0, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	a.drawScreenNotices(1024, 768)
	a.sprites.End()
	var newsPixel [4]byte
	width := a.chatTextFont.Width("Destroyed altar")
	gl.ReadPixels(int32(512-width/2+width+4), 768-(384-150+13)-1, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&newsPixel[0]))
	if newsPixel[0] == 0 && newsPixel[1] == 0 && newsPixel[2] == 0 {
		t.Fatal("news missing native bordered panel interior below the old flat rectangle")
	}
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if err := render.SavePNG(filepath.Join(dir, "altar-news.png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
	a.screenNotices = nil
	visual, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer visual.Delete()
	visual.View = render.LookAtLH(render.V3{0, 4, -12}, render.V3{0, 2, 0}, render.V3{0, 1, 0})
	visual.Proj = render.PerspectiveFovLH(math.Pi/3, 1024.0/768, .1, 100)
	visual.FogStart, visual.FogEnd = 100, 1000
	for country := range uint8(2) {
		a.wv = &worldView{pos: render.V3{0, 1, 0}}
		a.state.SelfID = 1
		a.state.Faction.Faction = country
		a.blessingFeedbackActive = false
		a.state.Blessing = game.BlessingState{Known: true, Country: country, Count: 12288, Maximum: 12288, RemainingMS: 600000}
		a.confirmedBlessing(proto.GXBlessingSnapshot)
		if len(a.wv.impacts) != 1 || a.wv.impacts[0].instance.Skipped != 0 {
			t.Fatalf("original blessing FX country=%d: %+v", country, a.wv.impacts)
		}
		a.frameDT = .1
		for range 40 {
			a.updateResolvedEffects()
		}
		if len(a.wv.impacts) != 1 {
			t.Fatal("burst ended before its authored eruption")
		}
		gl.ClearColor(0, 0, 0, 1)
		gl.ClearDepth(1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.wv.impacts[0].instance.Draw(visual)
		pixels := make([]byte, 1024*768*4)
		gl.ReadPixels(0, 0, 1024, 768, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		visible := 0
		for i := 0; i < len(pixels); i += 4 {
			if pixels[i] > 0 || pixels[i+1] > 0 || pixels[i+2] > 0 {
				visible++
			}
		}
		if visible < 100 {
			t.Fatalf("country %d eruption has no visible pixels: %d", country, visible)
		}
		if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
			if err := render.SavePNG(filepath.Join(dir, fmt.Sprintf("blessing-burst-%d.png", country)), 1024, 768); err != nil {
				t.Fatal(err)
			}
		}
		for range 20 {
			a.updateResolvedEffects()
		}
		if len(a.wv.impacts) != 1 {
			t.Fatal("ordinary five-second impact cutoff truncated blessing")
		}
		for range 80 {
			a.updateResolvedEffects()
		}
		if len(a.wv.impacts) != 0 {
			t.Fatal("authored blessing effect did not expire")
		}
		t.Logf("country %d local blessing eruption: %d visible pixels; full authored lifetime, no omitted emitters", country, visible)

	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
}
