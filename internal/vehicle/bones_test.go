package vehicle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBonesRejectInvalidBindings(t *testing.T) {
	for _, raw := range []string{"rider=1", "[0]\nrider=1\nrider=2", "[0]\npassenger=2", "[0]\nrider=3\npassenger=3", "[0]\nrider=1\n[0]\nrider=2", "[0]\nrider=1\nunknown=2"} {
		if _, err := ParseBones([]byte(raw)); err == nil {
			t.Fatal("accepted invalid bindings", raw)
		}
	}
	for _, race := range RaceFolders {
		raw, err := os.ReadFile(filepath.Join("../../resources/vehicle", race, "bones.ini"))
		if err != nil {
			t.Fatal(err)
		}
		bones, err := ParseBones(raw)
		if err != nil {
			t.Fatal(err)
		}
		if len(bones) != 8 || bones[0].Passenger != -1 || bones[1].Passenger < 0 || bones[6].Rider != 1 {
			t.Fatal("incomplete native seat table", race, bones)
		}
	}
}
