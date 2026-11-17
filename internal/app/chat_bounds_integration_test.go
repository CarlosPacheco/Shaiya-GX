//go:build integration

package app

import (
	"bytes"
	"fmt"
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
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestChatTextStaysInsidePanelsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("GL and external resources")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, e := platform.Open(platform.Config{Width: 1024, Height: 768, Hidden: true})
	if e != nil {
		t.Fatal(e)
	}
	defer win.Close()
	if e = render.Init(false); e != nil {
		t.Fatal(e)
	}
	arc, e := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if e != nil {
		t.Fatal(e)
	}
	defer arc.Close()
	a, e := New(Config{}, arc)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	draw := func(entries []ui.ChatEntry) []byte {
		gl.Viewport(0, 0, 1024, 768)
		gl.ClearColor(.15, .2, .25, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.chatUI.SetEntries(a.chatFont(), entries)
		a.sprites.Begin(1024, 768)
		a.chatUI.Draw(a.sprites, a.chatFont(), a.white, 768, false, "")
		a.sprites.End()
		pixels := make([]byte, 1024*768*4)
		gl.ReadPixels(0, 0, 1024, 768, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		return pixels
	}
	empty := draw(nil)
	// A single line belongs to the lowest complete cell in both histories.
	first := draw([]ui.ChatEntry{{Text: "First gjpqy", System: true, Color: [4]float32{1, 1, 1, 1}}, {Text: "First gjpqy", Color: [4]float32{1, 1, 1, 1}}})
	for _, bounds := range [][2]int{{420, 532}, {540, 684}} {
		changed := 0
		for y := bounds[0]; y < bounds[1]; y++ {
			off := ((767-y)*1024 + 25) * 4
			if !bytes.Equal(empty[off:off+316*4], first[off:off+316*4]) {
				if y < bounds[1]-16 {
					t.Fatal("first chat line floats above bottom row", y, bounds)
				}
				changed++
			}
		}
		if changed < 5 {
			t.Fatal("first chat line missing or clipped", bounds, changed)
		}
	}

	var entries []ui.ChatEntry
	for i := 0; i < 20; i++ {
		for _, system := range []bool{true, false} {
			entries = append(entries, ui.ChatEntry{Text: fmt.Sprintf("Line %02d: Ágjpqy UTF-8 日本語 😀", i), System: system, Color: [4]float32{1, 1, 1, 1}})
		}
	}
	full := draw(entries)
	for _, band := range [][2]int{{418, 420}, {532, 534}, {538, 540}, {684, 686}} {
		for y := band[0]; y < band[1]; y++ {
			offset := ((767-y)*1024 + 25) * 4
			if !bytes.Equal(empty[offset:offset+316*4], full[offset:offset+316*4]) {
				t.Fatal("chat glyphs escaped panel at y", y)
			}
		}
	}
	a.state = game.New()
	a.state.AddChat(game.ChatLine{Op: proto.ChatNormal, Sender: "Priest", Text: "Gear", Items: proto.ChatItems{
		{Type: 12, TypeID: 108, Count: 1, Gems: [6]uint8{164, 243}, Craft: "01020304050607080920"},
		{Type: 25, TypeID: 4, Count: 5},
	}})
	a.lastMouseX = 26 + a.chatFont().Width("Priest : Gear ")
	a.lastMouseY = 683 - a.chatFont().Height + 2
	gl.Clear(gl.COLOR_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	a.drawChat(1024, 768)
	if len(a.tooltip) == 0 || len(a.tooltipExtra.effects) != 1 {
		t.Fatalf("linked equipment hover lost its shared tooltip: %+v", a.tooltip)
	}
	a.drawSharedTooltip(1024, 768)
	a.sprites.End()
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if e = render.SavePNG(filepath.Join(dir, "chat-item-tooltip.png"), 1024, 768); e != nil {
			t.Fatal(e)
		}
	}
	if e = a.Preview("world-stream"); e != nil {
		t.Fatal(e)
	}
	slot := a.state.Slots[0]
	slot.Name = "Portrait Name"
	a.state.Slots[0] = slot
	for i := 0; i < 12; i++ {
		a.state.AddChat(game.ChatLine{Style: 0, Text: fmt.Sprintf("System line %02d stays inside the panel.", i)})
	}
	for frame := 0; frame < 500 && !a.SceneReady(); frame++ {
		a.Draw(1024, 768)
	}
	a.Draw(1024, 768)
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if e = os.MkdirAll(dir, 0755); e != nil {
			t.Fatal(e)
		}
		if e = render.SavePNG(filepath.Join(dir, "chat-clock-layout.png"), 1024, 768); e != nil {
			t.Fatal(e)
		}
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("chat GL error %#x", code)
	}
}
