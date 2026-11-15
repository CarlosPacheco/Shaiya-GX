package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/vehicle"
	"os"
	"path"
	"testing"
)

func TestRealVehicleSeatSkeletons(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("requires original assets")
	}
	arc, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	for race, code := range []string{"Hu", "El", "Vi", "De"} {
		raw, err := arc.ReadFile("Vehicle/Vehicle_" + code + "_01.MON")
		if err != nil {
			t.Fatal(err)
		}
		rows, err := model.ParseMON(raw)
		if err != nil {
			t.Fatal(err)
		}
		raw, err = os.ReadFile("../../resources/vehicle/bones.ini")
		if err != nil {
			t.Fatal(err)
		}
		bones, err := vehicle.ParseBones(raw)
		if err != nil {
			t.Fatal(err)
		}
		for index, b := range bones[uint8(race)] {
			row := rows[index]
			for _, part := range row.Parts {
				raw, err = arc.ReadFile("Vehicle/3DC/" + path.Base(part[0]))
				if err != nil {
					t.Fatal(err)
				}
				mesh, err := model.Parse3DC(raw)
				if err != nil {
					t.Fatal(err)
				}
				if b.Rider >= len(mesh.Bones) || b.Passenger >= len(mesh.Bones) {
					t.Fatalf("race %s MON %d seats outside %d bones", code, index, len(mesh.Bones))
				}
			}
		}
	}
}
