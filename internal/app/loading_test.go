package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/session"
)

func loadingTipArchive(t *testing.T, content string) *archive.Archive {
	t.Helper()
	dir := t.TempDir()
	if content != "" {
		if err := os.WriteFile(filepath.Join(dir, "tip.ini"), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	arc, err := archive.OpenDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = arc.Close() })
	return arc
}

func TestLoadingTipStaysUntilNextScreen(t *testing.T) {
	a := &App{state: game.New(), world: &session.World{}, arc: loadingTipArchive(t, "[Tips]\n1=Primera misión.\n")}
	a.state.Phase, a.state.MapID = game.PhaseLoading, 1
	if !a.updateMapLoading() || a.loading.tip != "Primera misión." {
		t.Fatalf("first loading screen tip = %q", a.loading.tip)
	}

	a.arc = loadingTipArchive(t, "[Tips]\n12=Segundo mapa.\n")
	if !a.updateMapLoading() || a.loading.tip != "Primera misión." {
		t.Fatalf("tip changed during the same load: %q", a.loading.tip)
	}

	a.loading.started = true
	a.wv = &worldView{mapID: 1}
	a.state.MapID = 2
	// A transfer that supersedes the load restarts the plan but keeps the same
	// loading screen (picture and tip), so it is not seen as a second screen.
	if !a.updateMapLoading() || a.loading.tip != "Primera misión." || a.loading.started || a.loading.mapID != 2 {
		t.Fatalf("superseded load changed its screen: %+v", a.loading)
	}
}

func TestCharacterSelectionLoadingTip(t *testing.T) {
	arc := loadingTipArchive(t, "[Tips]\n# ignored\n1=First\n42=Second\n43=\n[Other]\n1=ignored\n")
	a := &App{state: game.New(), world: &session.World{}, screen: screenCharacters, arc: arc}
	a.state.Phase = game.PhaseCharSelect
	if !a.updateMapLoading() || !a.loading.selection {
		t.Fatal("character selection did not start loading")
	}
	if got := a.loading.tip; got != "First" && got != "Second" {
		t.Fatalf("character selection displayed an unconfigured tip: %q", got)
	}
}

func TestLoadingWithoutTipsShowsNoMessage(t *testing.T) {
	for _, content := range []string{"", "[Tips]\n1=\n2=\" \"\n"} {
		a := &App{arc: loadingTipArchive(t, content)}
		a.beginMapLoading(1, false)
		if a.loading.tip != "" {
			t.Fatalf("unexpected tip: %q", a.loading.tip)
		}
	}
}

func TestLogoutDiscardsPreviousCharacterLoadingMap(t *testing.T) {
	a := inputApp()
	a.sprites = &render.Sprites{}
	a.wv = nil
	a.world = &session.World{}
	a.screen = screenWorld
	a.state.Phase, a.state.SelfID, a.state.MapID = game.PhaseInWorld, 20, 64
	a.beginMapLoading(64, false)

	a.applyWorldPacket(proto.NewWriter(proto.Logout).Bytes())
	if a.screen != screenCharacters || a.loading.active || a.state.MapID != 0 {
		t.Fatalf("logout retained the island: screen=%d loading=%+v map=%d", a.screen, a.loading, a.state.MapID)
	}
	if !a.updateMapLoading() || !a.loading.selection || a.loading.mapID != 0 {
		t.Fatalf("roster did not get its own loading screen: %+v", a.loading)
	}
}
