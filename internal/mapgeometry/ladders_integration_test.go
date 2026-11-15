//go:build integration

package mapgeometry

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"os"
	"reflect"
	"testing"
)

func TestOriginalLadderCatalogMatchesLooseAssets(t *testing.T) {
	directory := os.Getenv("SHAIYA_DATA")
	if directory == "" {
		t.Skip("requires original loose assets")
	}
	arc, err := archive.OpenDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	for id, want := range nativeLadders {
		raw, err := arc.ReadFile(fmt.Sprintf("world/%d.wld", id))
		if err != nil {
			t.Fatal(err)
		}
		w, err := world.ParseWLD(raw)
		if err != nil {
			t.Fatal(err)
		}
		got, err := BuildLadders(arc.ReadFile, w)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("map %d ladder catalog differs from original asset placements", id)
		}
	}
}
