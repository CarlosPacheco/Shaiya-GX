//go:build integration

package app

import (
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestLocalBotsCommandsAndMapLifecycleGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires OpenGL and client data")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	win, err := platform.Open(platform.Config{Title: "Local PvP commands", Width: 640, Height: 360, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(false); err != nil {
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
	a.state.Self.IsGM = true
	self, entities := a.state.Self, len(a.state.Entities)
	submit := func(line string) {
		t.Helper()
		a.chatOpen = true
		a.chatInput = ui.TextField{Text: line}
		if !a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}) || a.chatOpen {
			t.Fatalf("command escaped local chat handling: %s", line)
		}
	}
	spawn := func(line string, count int) {
		t.Helper()
		submit(line)
		for range 6 {
			a.frameDT = 1.0 / 30
			a.Draw(640, 360)
		}
		if a.wv.bots == nil || a.wv.bots.Count() != count {
			t.Fatalf("%s failed to create %d bots", line, count)
		}
		if a.world != nil || len(a.state.Entities) != entities || a.state.Self != self {
			t.Fatalf("%s modified the authoritative character or entities", line)
		}
	}
	spawn("/bots 6", 6)
	spawn("/bots 1", 1)
	submit("/bots off")
	if a.wv.bots != nil {
		t.Fatal("off left a simulation running")
	}
	spawn("/bots 2", 2)
	a.state.Self.IsGM = false
	a.Draw(640, 360)
	if a.wv.bots != nil {
		t.Fatal("simulation survived loss of GM permission")
	}
	a.state.Self.IsGM = true
	spawn("/bots 3", 3)
	old := a.wv
	a.releaseWorld()
	if old.bots != nil || a.wv != nil {
		t.Fatal("simulation survived world release")
	}
}
