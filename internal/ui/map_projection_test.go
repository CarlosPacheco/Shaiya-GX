package ui

import (
	"math"
	"testing"
)

func TestDungeonProjectionCroppedCoordinatesAndIndependentScales(t *testing.T) {
	p := MapProjection{Width: 512 / float32(1.664), Height: 512 / float32(1.472)}
	for _, point := range [][2]float32{{0, 0}, {136.27, 229.12186}, {p.Width, p.Height}} {
		u, v, ok := p.Position(point[0], point[1])
		if !ok {
			t.Fatal("dungeon rejected", point)
		}
		view := p.Window(point[0], point[1], 100)
		fx, fy, ok := MapPoint(u, v, view)
		if !ok {
			t.Fatal("clamped window lost actor", view, point)
		}
		x, z, ok := p.Coordinates(view, fx, fy)
		if !ok || math.Abs(float64(x-point[0])) > .0001 || math.Abs(float64(z-point[1])) > .0001 {
			t.Fatal("calibrated map click mismatch", point, x, z, ok)
		}
		dx, dz := (view.U1-view.U0)*p.Width, (view.V1-view.V0)*p.Height
		if math.Abs(float64(dx-100)) > .0001 || math.Abs(float64(dz-100)) > .0001 {
			t.Fatal("world diameter stretched", dx, dz)
		}
	}
	if _, _, ok := (MapProjection{}).Position(1, 1); ok {
		t.Fatal("missing calibration invented coordinates")
	}
	if _, _, ok := p.Coordinates(UV{U1: 1, V1: 1}, -.1, .5); ok {
		t.Fatal("off-image click accepted")
	}
}
