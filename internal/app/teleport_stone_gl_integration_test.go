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
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestTeleportStonePanelGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("GL and external resources")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 1024, Height: 768, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	data, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer data.Close()
	a, err := New(Config{}, data)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.teleportStone.open = true
	a.teleportStone.points[0] = proto.TeleportPoint{Saved: true, Map: 1, X: 578, Z: 1762}
	a.teleportStone.points[3] = proto.TeleportPoint{Saved: true, Map: 18, X: 321, Z: 811}
	gl.Viewport(0, 0, 1024, 768)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	a.drawTeleportStone(1024, 768)
	a.sprites.End()
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("panel GL error %#x", code)
	}
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if err := render.SavePNG(filepath.Join(dir, "recorded-points.png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}

	a.state = game.New()
	request := proto.NewWriter(proto.GXSummonRune)
	request.U8(proto.SummonOffer)
	request.U32(101)
	request.U32(2)
	request.U16(1)
	request.U16(60000)
	request.Fixed("Priest", 21, true)
	a.confirmedSummonRune(request.Bytes())
	if !a.summonMessage.Visible || a.summonRune.expires != a.clock+60 {
		t.Fatal("summon confirmation did not open for 60 seconds")
	}
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	a.summonMessage.Draw(a.sprites, a.font, 1024, 768)
	a.sprites.End()
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if err := render.SavePNG(filepath.Join(dir, "summon-confirmation.png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}

}
