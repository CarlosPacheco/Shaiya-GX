package vehicle

import (
	"os"

	"testing"
)

func TestBonesRejectInvalidBindings(t *testing.T) {
	for _, raw := range []string{"0=1", "[Human]\n0=1\n0=2", "[Human]\n0=-1", "[Human]\n0=3,3", "[Human]\n0=1\n[Human]\n1=2", "[Unknown]\n0=1", "[Human]\n256=1", "[Human]\n0=1,2,3"} {
		if _, err := ParseBones([]byte(raw)); err == nil {
			t.Fatal("accepted invalid bindings", raw)
		}
	}
	raw, err := os.ReadFile("../../resources/vehicle/bones.ini")
	if err != nil {
		t.Fatal(err)
	}
	all, err := ParseBones(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 4 {
		t.Fatal("missing race sections")
	}
	for race := range RaceNames {
		bones := all[uint8(race)]
		if len(bones) != 8 || bones[0].Passenger != -1 || bones[1].Passenger < 0 || bones[6].Rider != 1 {
			t.Fatal("incomplete native seat table", race, bones)
		}
	}
}
