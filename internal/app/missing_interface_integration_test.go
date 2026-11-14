//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestMissingInterfaceLoginGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires OpenGL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 640, Height: 480, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, "interface"), 0700); err != nil {
		t.Fatal(err)
	}
	// One damaged atlas and four absent atlases must all recover.
	if err := os.WriteFile(filepath.Join(directory, "interface/login.png"), []byte("damaged PNG"), 0600); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := &App{arc: arc, cache: map[string]*render.Texture{}}
	defer func() {
		for _, texture := range a.cache {
			texture.Delete()
		}
	}()
	login, err := ui.NewLogin(a)
	if err != nil {
		t.Fatal(err)
	}
	if login == nil || login.OK == nil || login.Cancel == nil || a.missingInterfaces != 5 {
		t.Fatalf("login did not recover all artwork: %d", a.missingInterfaces)
	}
	first, err := a.Interface("LOGIN.PNG")
	if err != nil {
		t.Fatal(err)
	}
	second, err := a.Interface("login.png")
	if err != nil {
		t.Fatal(err)
	}
	if first != second || a.missingInterfaces != 5 || first.W != 256 || first.H != 256 || first.ID == 0 {
		t.Fatal("placeholder was not uploaded and cached once")
	}
}
