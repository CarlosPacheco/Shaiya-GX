package model

import (
	"os"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
)

// Creature data is normalized offline. Runtime Parse3DC remains unchanged.
func TestRealCreature3DC(t *testing.T) {
	if os.Getenv("SHAIYA_DATA") == "" {
		t.Skip("requires normalized original creature data")
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	count := 0
	arc.Walk(func(file string, entry *archive.File) {
		lower := strings.ToLower(file)
		if !strings.HasSuffix(lower, ".3dc") || !(strings.HasPrefix(lower, "monster/") || strings.HasPrefix(lower, "npc/")) {
			return
		}
		raw, err := arc.Read(entry)
		if err != nil {
			t.Error(file, err)
			return
		}
		mesh, err := Parse3DC(raw)
		if err != nil {
			t.Error(file, err)
			return
		}
		if len(mesh.Vertices) == 0 || len(mesh.Faces) == 0 {
			t.Error(file, "empty creature mesh")
			return
		}
		count++
	})
	if count == 0 {
		t.Fatal("no creature meshes audited")
	}
	t.Logf("%d monster/NPC meshes accepted by unchanged runtime parser", count)
}
