package scene

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
)

func TestSparseInterfaceTextureOverlay(t *testing.T) {
	root := t.TempDir()
	texture := func(width int) []byte {
		t.Helper()
		var data bytes.Buffer
		if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, width, 1))); err != nil {
			t.Fatal(err)
		}
		return data.Bytes()
	}
	for name, data := range map[string][]byte{
		"interface/Buttons/Start.png":                          texture(1),
		"interface/Buttons/Exit.png":                           texture(1),
		"interface/Buttons/Back.png":                           texture(1),
		"Languages/interface/interface_es/Buttons/Start.png":   texture(2),
		"Languages/interface/interface_es/Buttons/Back.png":    []byte("invalid PNG"),
		"Languages/interface/interface_pt-BR/Buttons/Exit.png": texture(3),
	} {
		file := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	arc, err := archive.OpenDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()

	for _, tc := range []struct {
		language, file string
		width          int
	}{
		{"", "interface/Buttons/Start.png", 1},
		{"es", "Interface/buttons/start.PNG", 2},
		{"es", "interface/Buttons/Exit.png", 1},
		{"es", "interface/Buttons/Back.png", 1},
		{"pt-BR", "interface/Buttons/Exit.png", 3},
		{"pt-BR", "interface/Buttons/Start.png", 1},
		{"missing", "interface/Buttons/Start.png", 1},
	} {
		assets := NewAssets(arc)
		assets.UseInterfaceLanguage(tc.language)
		got, err := assets.ReadTexture(tc.file)
		if err != nil || got.RGBA == nil || got.RGBA.W != tc.width {
			t.Errorf("%s %s: got %+v, %v, want width %d", tc.language, tc.file, got, err, tc.width)
		}
	}
}
