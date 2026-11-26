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

func TestLoadingTipTranslationFallsBackToEnglish(t *testing.T) {
	root := t.TempDir()
	tips := filepath.Join(root, "Languages", "Tips")
	if err := os.MkdirAll(tips, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Tip.INI"), []byte("[Tips]\n1=Welcome\n"), 0644); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := &App{arc: arc}
	for _, tc := range []struct{ content, want string }{
		{"", "Welcome"},
		{"[Tips]\n2=Otro consejo\n", "Welcome"},
		{"[Tips]\n1=\n", "Welcome"},
		{"[Tips]\n1=Bienvenido\n", "Bienvenido"},
	} {
		if tc.content != "" {
			if err := os.WriteFile(filepath.Join(tips, "tip_es.ini"), []byte(tc.content), 0644); err != nil {
				t.Fatal(err)
			}
		}
		a.loadLocalizedLoadingTips(2)
		if got := a.randomLoadingTip(); got != tc.want {
			t.Fatalf("tip for %q = %q, want %q", tc.content, got, tc.want)
		}
	}
	a.loadLocalizedLoadingTips(1)
	if got := a.randomLoadingTip(); got != "Welcome" {
		t.Fatalf("switching to English kept %q", got)
	}
}
