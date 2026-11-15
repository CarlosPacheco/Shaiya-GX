package collision

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

func TestOriginalTriangleOracle(t *testing.T) {
	var data struct {
		Fixtures []struct {
			Origin, Velocity Vec
			Triangle         Triangle
			Hit              bool
			Distance         float64
			Point            Vec
		}
	}
	raw, err := os.ReadFile("testdata/native_triangles.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	for i, f := range data.Fixtures {
		actual := SweepTriangle(f.Origin, f.Velocity, f.Triangle)
		if actual.Hit != f.Hit || f.Hit && (math.Abs(actual.Distance-f.Distance) > .0001 || actual.Point.Sub(f.Point).Len() > .0001) {
			t.Errorf("native query %d: got %+v want %+v", i, actual, f)
		}
	}
}

func TestOriginalMovementOracle(t *testing.T) {
	var data struct {
		Fixtures []struct {
			Origin, Velocity Vec
			Triangles        []Triangle
			Result           Vec
		}
	}
	raw, err := os.ReadFile("testdata/native_movement.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	for i, f := range data.Fixtures {
		visit := func(_ Box, emit func(Triangle)) {
			for _, triangle := range f.Triangles {
				emit(triangle)
			}
		}
		actual, _ := Move(f.Origin, f.Velocity, visit)
		if actual.Sub(f.Result).Len() > .0001 {
			t.Errorf("native motion %d: got %v want %v", i, actual, f.Result)
		}
	}
}
