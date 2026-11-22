package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
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
	if !a.updateMapLoading() || a.loading.tip != "Segundo mapa." || a.loading.started {
		t.Fatalf("new load did not select a new tip: %+v", a.loading)
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
