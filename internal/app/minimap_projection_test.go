package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"math"
	"testing"
)

func TestDungeonMinimapMarkersAndGMAgreeWithImage(t *testing.T) {
	a := inputApp()
	a.wv.mapID = 15
	a.wv.world = &scene.World{WLD: &world.WLD{Dungeon: true}}
	v := a.mapState()
	v.image = &render.Texture{W: 512, H: 512}
	a.dungeonMinimaps = dungeonMinimapData{loaded: true, scales: map[uint16]world.DungeonMinimap{15: {X: 1.664, Z: 1.472}}, sizes: map[uint16]float32{15: 328}}
	p := a.mapProjection(v)
	x, z := float32(136.27), float32(229.12186)
	u, imageV, ok := p.Position(x, z)
	if !ok {
		t.Fatal("player off minimap")
	}
	square := max(p.Width, p.Height)
	carrierU, carrierV, _ := ui.MapPosition(x, z, square)
	marks := calibrateMapMarkers([]ui.MapMarker{{U: carrierU, V: carrierV, Kind: ui.MapParty}}, p, square)
	if len(marks) != 1 || math.Abs(float64(marks[0].U-u)) > .00001 || math.Abs(float64(marks[0].V-imageV)) > .00001 {
		t.Fatal("party/quest/relic projection differs", marks, u, imageV)
	}
	for _, view := range []ui.UV{{U1: 1, V1: 1}, p.Window(x, z, 100)} {
		fx, fy, _ := ui.MapPoint(u, imageV, view)
		requestU, requestV, ok := a.mapTeleportCoordinates(15, view, fx, fy)
		if !ok || math.Abs(float64(requestU*328-x)) > .0001 || math.Abs(float64((1-requestV)*328-z)) > .0001 {
			t.Fatal("GM teleport differs from drawn coordinate", requestU, requestV, x, z)
		}
	}
	if _, _, ok := a.mapTeleportCoordinates(15, ui.UV{U1: 1, V1: 1}, 1, 0); ok {
		t.Fatal("image outside actual map accepted")
	}
	a.wv.world.WLD = &world.WLD{MapSize: 2048}
	if p := a.mapProjection(v); p.Width != 2048 || p.Height != 2048 {
		t.Fatal("dungeon CFG altered field mapping", p)
	}
	delete(a.dungeonMinimaps.scales, 15)
	a.wv.world.WLD.Dungeon = true
	if p := a.mapProjection(v); p.Width != 0 {
		t.Fatal("missing CFG invented a dungeon projection", p)
	}
}
func TestNonSquareMarkersKeepLongAxisAndCachedSource(t *testing.T) {
	p := ui.MapProjection{Width: 100, Height: 200}
	cached := []ui.MapMarker{{U: .25, V: .25, Kind: ui.MapQuestStart}, {U: .75, V: .25}}
	marks := append([]ui.MapMarker(nil), cached...)
	marks = calibrateMapMarkers(marks, p, 200)
	if len(marks) != 1 || marks[0].U != .5 || marks[0].V != .25 || cached[0].U != .25 {
		t.Fatal("long-axis marker lost or cached marker mutated", marks, cached)
	}
}

func TestNativeMinimapZoomScalesWithMap(t *testing.T) {
	for _, p := range []ui.MapProjection{{Width: 1024, Height: 1024}, {Width: 4096, Height: 2048}} {
		x, z := p.Width/2, p.Height/2
		view := p.MinimapWindow(x, z, 1)
		if view.U1-view.U0 != 136.0/512 || view.V1-view.V0 != 136.0/512 {
			t.Fatalf("unexpected default coverage for %+v: %+v", p, view)
		}
		// The player is the anchor pixel of the circle, not its centre.
		fx, fy := float32(ui.MinimapAnchorX-ui.MinimapCircleEdge)/ui.MinimapImageSize, float32(ui.MinimapAnchorY-ui.MinimapCircleEdge)/ui.MinimapImageSize
		gotX, gotZ, ok := p.Coordinates(view, fx, fy)
		if !ok || math.Abs(float64(gotX-x)) > .05*float64(p.Width)/512 || math.Abs(float64(gotZ-z)) > .05*float64(p.Height)/512 {
			t.Fatal("minimap click and image projection disagreed", gotX, gotZ, ok)
		}
		// An edge position keeps the anchor fixed and slides the window past the map.
		edge := p.MinimapWindow(0, 0, 1)
		if edge.U0 >= 0 || edge.V1 <= 1 {
			t.Fatal("the window was clamped at the map corner", edge)
		}
	}
}

func TestMinimapPositionIsTruncatedToTheNativeGrid(t *testing.T) {
	p := ui.MapProjection{Width: 1024, Height: 1024}
	a, b := p.MinimapWindow(100.9, 100.9, 1), p.MinimapWindow(100.1, 100.1, 1)
	if a != b {
		t.Fatal("positions inside one 1/512 cell moved the map", a, b)
	}
}
