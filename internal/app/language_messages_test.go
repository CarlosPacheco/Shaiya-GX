package app

import (
	"encoding/binary"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"unicode/utf16"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestContentMessageFallback(t *testing.T) {
	const basePath = "sysmsg-uni.txt"
	const spanishPath = "Languages/Messages/sysmgtxt-es.txt"
	base := []byte{0xff, 0xfe}
	for _, unit := range utf16.Encode([]rune("1\t\"You died\"\n2\t\"<p> died\"\n3\t\"Inventory full\"\n4\t\"Line one\\nLine two\"\n5\t\"Equip <i>\"\n6\t\"Declined\"\n7\t\"English description.\"\n")) {
		base = binary.LittleEndian.AppendUint16(base, unit)
	}

	for _, tc := range []struct {
		name  string
		id    int
		paths []string
		want  string
	}{
		{"English", 1, []string{basePath}, "You died"},
		{"unknown", 99, []string{basePath}, "You died"},
		{"missing Vietnamese", 10, []string{basePath, "Languages/Messages/sysmgtxt-vi.txt"}, "You died"},
		{"partial Spanish", 2, []string{basePath, spanishPath}, "Has muerto"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var paths []string
			english, got := loadContentMessages(func(path string) ([]byte, error) {
				paths = append(paths, path)
				switch path {
				case basePath:
					return base, nil
				case spanishPath:
					return []byte("\ufeff1\t\"Has muerto\"\n2\t\"Ha muerto\"\n3\t\"\"\n4\t\"Línea sola\"\n5\t\"<i>, equipar\"\n6\t\"Sin cierre"), nil
				default:
					return nil, fs.ErrNotExist
				}
			}, tc.id)
			if !slices.Equal(paths, tc.paths) {
				t.Fatalf("read paths %q, want %q", paths, tc.paths)
			}
			if english.Get(1) != "You died" || got.Get(1) != tc.want {
				t.Fatalf("English = %q, selected = %q", english.Get(1), got.Get(1))
			}
			if got.Get(2) != "<p> died" || got.Get(3) != "Inventory full" ||
				got.Get(4) != "Line one\nLine two" || got.Get(6) != "Declined" ||
				got.Get(7) != "English description." {
				t.Fatal("invalid or empty translations replaced English")
			}
			if tc.id == 2 && got.Format(5, map[string]string{"i": "Lapis"}) != "Lapis, equipar" {
				t.Fatal("translated placeholder did not expand")
			}
		})
	}
}

func TestReloadContentMessagesKeepsChatHistory(t *testing.T) {
	root := t.TempDir()
	assets, err := archive.OpenDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	defer assets.Close()

	a := inputApp()
	a.arc = assets
	a.englishMsg = text.SysMsg{1: "You died.", 2: "English description.", 353: "[Party] <p>: <b>"}
	a.msg = a.englishMsg
	addChat := func(body string) {
		a.state.AddChat(game.ChatLine{Op: proto.ChatParty, Sender: "Alice", Text: body})
	}
	addChat("before")

	dir := filepath.Join(root, "Languages", "Messages")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "sysmgtxt-es.txt")
	writeSpanish := func(message string) {
		t.Helper()
		if err := os.WriteFile(file, []byte("1\t\""+message+"\"\n353\t\"[Grupo] <p>: <b>\"\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	writeSpanish("Moriste.")
	a.reloadContentMessages(2)
	if a.tr(1) != "Moriste." || a.tr(2) != "English description." || a.interfaceText(1) != "You died." {
		t.Fatal("live translation or English fallback was not applied")
	}
	addChat("after")
	writeSpanish("Tu personaje ha muerto.")
	a.reloadContentMessages(2)
	if a.tr(1) != "Tu personaje ha muerto." {
		t.Fatal("an edit to the selected catalog was not reloaded")
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	a.reloadContentMessages(2)
	if a.tr(1) != "You died." {
		t.Fatal("removing the selected catalog did not restore English")
	}
	addChat("later")
	entries := a.formattedChat()
	got := []string{entries[0].Text, entries[1].Text, entries[2].Text}
	want := []string{"[Party] Alice: before", "[Grupo] Alice: after", "[Party] Alice: later"}
	if !slices.Equal(got, want) {
		t.Fatalf("chat history = %q, want %q", got, want)
	}
}
