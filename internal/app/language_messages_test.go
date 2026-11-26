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
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
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
		{"removed Vietnamese falls back to English", 10, []string{basePath}, "You died"},
		{"missing Portuguese", 3, []string{basePath, "Languages/Messages/sysmgtxt-pt-BR.txt"}, "You died"},
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

func TestRankedSkillTranslationsAndEnglishFallback(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "Languages", "skills")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "skills_es.csv")
	if err := os.WriteFile(path, []byte("id,level,name,description\n31,1,Hechizo,Inflige 10 de daño.\n31,2,,Inflige 20 de daño.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()

	a := inputApp()
	a.arc, a.assets = arc, scene.NewAssets(arc)
	first := sdata.SkillKey{ID: 31, Level: 1}
	second := sdata.SkillKey{ID: 31, Level: 2}
	third := sdata.SkillKey{ID: 31, Level: 3}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{
		first:  {Name: "Spell 1", Description: "Deals 10 damage.", MP: 4},
		second: {Name: "Spell 2", Description: "Deals 20 damage.", MP: 5},
		third:  {Name: "Spell 3", Description: "Deals 30 damage.", MP: 6},
	}
	a.loadLanguage(2)
	if d, _ := a.skillDefinition(first.ID, first.Level); d.Name != "Hechizo" || d.Description != "Inflige 10 de daño." || d.MP != 4 {
		t.Fatalf("rank 1: %+v", d)
	}
	if d, _ := a.skillDefinition(second.ID, second.Level); d.Name != "Spell 2" || d.Description != "Inflige 20 de daño." || d.MP != 5 {
		t.Fatalf("rank 2 with partial translation: %+v", d)
	}
	if d, _ := a.skillDefinition(third.ID, third.Level); d != a.skillDefs[third] {
		t.Fatalf("missing rank 3: %+v", d)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	arcWithoutSpanish, err := archive.OpenDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	defer arcWithoutSpanish.Close()
	withoutSpanish := &App{arc: arcWithoutSpanish, assets: scene.NewAssets(arcWithoutSpanish), skillDefs: a.skillDefs}
	withoutSpanish.loadLanguage(2)
	if d, _ := withoutSpanish.skillDefinition(first.ID, first.Level); d != a.skillDefs[first] {
		t.Fatalf("missing Spanish catalog: %+v", d)
	}
}
