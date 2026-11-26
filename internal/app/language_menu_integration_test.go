//go:build integration

package app

import (
	"bytes"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestLanguageMenuGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
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
	a.screen = screenCharacters
	a.openGameMenu()
	a.gameMenu.Open(ui.MenuUser)
	x, y := a.gameMenu.Origin(1024, 768)
	click := func(px, py int) {
		in := platform.Input{MouseX: x + px, MouseY: y + py, Pressed: true, MouseDown: true}
		a.updateGameMenu(&in, 1024, 768)
		in.Pressed, in.MouseDown, in.Clicked = false, false, true
		a.updateGameMenu(&in, 1024, 768)
	}
	click(265, 55)
	if !a.gameMenu.PopupOpen() {
		t.Fatal("language dropdown did not open")
	}
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		gl.Viewport(0, 0, 1024, 768)
		gl.ClearColor(.12, .14, .16, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(1024, 768)
		a.gameMenu.Draw(a.sprites, a.font, 1024, 768, a.white)
		a.sprites.End()
		if err := render.SavePNG(filepath.Join(dir, "shaiya-gx-language.png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
	click(265, 109) // Español
	if a.gameMenu.PopupOpen() || a.gameMenu.Settings.LanguageID != 2 {
		t.Fatal("Español was not selected")
	}
	click(310, 404) // Cancel
	if a.gameMenu.Settings.LanguageID != ui.EnglishLanguageID {
		t.Fatal("Cancel did not restore English")
	}
	a.chars.SetChars(map[uint8]proto.CharacterSlot{0: {ID: 1, Name: "Test", Level: 10, MapID: 1}})
	a.servers.Servers = []proto.Server{{ID: 1, Name: "Test", Status: 2}}
	draw := func(screen func(*render.Sprites, *ui.Font, int, int)) []byte {
		gl.Viewport(0, 0, 1024, 768)
		gl.Clear(gl.COLOR_BUFFER_BIT)
		a.sprites.Begin(1024, 768)
		screen(a.sprites, a.font, 1024, 768)
		a.sprites.End()
		pixels := make([]byte, 1024*768*4)
		gl.ReadPixels(0, 0, 1024, 768, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		return pixels
	}
	englishChars, englishServers := draw(a.chars.Draw), draw(a.servers.Draw)
	a.gameMenu.Open(ui.MenuUser)
	click(265, 55)
	click(265, 109) // Español
	click(245, 404) // OK
	if a.msg.Get(5065) != a.englishMsg.Get(5065) && bytes.Equal(draw(a.chars.Draw), englishChars) {
		t.Fatal("character selection kept rendering English")
	}
	if a.msg.Get(10111) != a.englishMsg.Get(10111) && bytes.Equal(draw(a.servers.Draw), englishServers) {
		t.Fatal("server selection kept rendering English")
	}
	a.gameMenu.Open(ui.MenuUser)
	click(265, 55)
	click(265, 138) // Português (Brasil)
	click(245, 404) // OK
	if loadClientLanguage() != 3 {
		t.Fatal("Portuguese did not persist with layout saving disabled")
	}
	if want := translatedMessages(a.englishMsg, arc.ReadLiveFile, 3); !maps.Equal(a.msg, want) {
		t.Fatal("the chosen catalog was not applied when OK was pressed")
	}
}
