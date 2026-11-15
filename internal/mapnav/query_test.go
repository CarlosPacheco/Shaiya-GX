package mapnav

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"os"
	"path/filepath"
	"testing"
)

func floor(g *mapgeometry.Geometry, x0, z0, x1, z1, y float64) {
	a, b, c, d := collision.Vec{x0, y, z0}, collision.Vec{x1, y, z0}, collision.Vec{x0, y, z1}, collision.Vec{x1, y, z1}
	g.Add(collision.Triangle{a, c, b})
	g.Add(collision.Triangle{d, b, c})
}
func baked(t *testing.T, g *mapgeometry.Geometry, masks ...WalkMask) *Map {
	t.Helper()
	if os.Getenv("GX_RECAST_BAKER") == "" {
		t.Skip("requires the pinned offline Recast baker")
	}
	directory := t.TempDir()
	if err := Bake(g, 3, directory, masks...); err != nil {
		t.Fatal(err)
	}
	nav, err := Load(directory, 3)
	if err != nil {
		t.Fatal(err)
	}
	return nav
}
func TestLayeredPathWallsAndDisconnectedFloors(t *testing.T) {
	g := &mapgeometry.Geometry{}
	g.Reindex()
	floor(g, 0, 0, 20, 20, 0)
	floor(g, 0, 0, 20, 20, 5)
	// A wall across most of the lower floor, with a passage beyond Z=15.
	for _, tri := range collision.BlockerBox(collision.Vec{9, 0, 0}, collision.Vec{10, 3, 15}, 0) {
		g.Add(tri)
	}
	nav := baked(t, g)
	low, ok := nav.Project(Point{3, .2, 5}, .5, 1)
	if !ok || low[1] != 0 {
		t.Fatalf("lower floor %v %v", low, ok)
	}
	high, ok := nav.Project(Point{3, 5.2, 5}, .5, 1)
	if !ok || high[1] != 5 {
		t.Fatalf("upper floor %v %v", high, ok)
	}
	if _, ok = nav.Path(low, high); ok {
		t.Fatal("disconnected floors must not connect")
	}
	goal := Point{15, 0, 5}
	path, ok := nav.Path(low, goal)
	if !ok {
		t.Fatal("doorway route not found")
	}
	if nav.Visible(low, goal) {
		t.Fatal("attack line goes through a wall")
	}
	detour := false
	for _, p := range path {
		if p[2] > 15 {
			detour = true
		}
	}
	if !detour {
		t.Fatalf("route cuts wall: %v", path)
	}
	pos := low
	for i := 0; i < 500 && Distance(pos, goal) > .05; i++ {
		for len(path) > 0 && Distance(pos, path[0]) < .05 {
			path = path[1:]
		}
		next, _, consumed, ok := nav.Step(pos, path, 1, func(x, z float32) bool { return true })
		if !ok {
			t.Fatalf("route stalled at %v, path %v", pos, path)
		}
		if next[1] != 0 {
			t.Fatalf("changed floor: %v", next)
		}
		if next[0] > 8.8 && next[0] < 10.2 && next[2] < 15 {
			t.Fatalf("entered wall: %v", next)
		}
		pos = next
		path = path[consumed:]
	}
	if Distance(pos, goal) > .1 {
		t.Fatalf("did not reach goal: %v", pos)
	}
	if _, _, _, ok = nav.Step(low, []Point{{4, 0, 5}}, 1, func(x, z float32) bool { return false }); ok {
		t.Fatal("ignored server mask")
	}
}
func TestTerrainAndGenerationIntegrity(t *testing.T) {
	g := &mapgeometry.Geometry{MapSize: 32, Heights: make([]uint16, 17*17)}
	for i := range g.Heights {
		g.Heights[i] = 10000 + uint16(i%17*5)
	}
	g.Reindex()
	nav := baked(t, g)
	p, ok := nav.Project(Point{5, .25, 5}, .5, 1)
	if !ok || abs(p[1]-g.Height(p[0], p[2])) > .001 {
		t.Fatalf("outdoor terrain support: %v %v", p, ok)
	}
	dir := t.TempDir()
	if err := Bake(g, 3, dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "3.gxcol"), []byte("damaged"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir, 3); err == nil {
		t.Fatal("accepted corrupt collision generation")
	}
}

func TestClearanceStepsAndMissingGround(t *testing.T) {
	t.Run("low ceiling", func(t *testing.T) {
		g := &mapgeometry.Geometry{}
		g.Reindex()
		floor(g, 0, 0, 20, 20, 0)
		floor(g, 0, 0, 20, 20, 1.5)
		nav := baked(t, g)
		if _, ok := nav.Project(Point{5, 0, 5}, .1, .25); ok {
			t.Fatal("agent fits under a ceiling below its height")
		}
	})
	t.Run("gap", func(t *testing.T) {
		g := &mapgeometry.Geometry{}
		g.Reindex()
		floor(g, 0, 0, 8, 20, 0)
		floor(g, 12, 0, 20, 20, 0)
		nav := baked(t, g)
		if _, ok := nav.Path(Point{5, 0, 5}, Point{15, 0, 5}); ok {
			t.Fatal("partial path across missing ground accepted")
		}
	})
	for _, height := range []float64{.3, 1} {
		t.Run(fmt.Sprint(height), func(t *testing.T) {
			g := &mapgeometry.Geometry{}
			g.Reindex()
			floor(g, 0, 0, 10, 20, 0)
			floor(g, 10, 0, 20, 20, height)
			nav := baked(t, g)
			_, ok := nav.Path(Point{5, 0, 5}, Point{15, float32(height), 5})
			if ok != (height < .5) {
				t.Fatalf("step %g route=%v", height, ok)
			}
		})
	}
}

func TestBakedServerMaskDetour(t *testing.T) {
	g := &mapgeometry.Geometry{}
	g.Reindex()
	floor(g, 0, 0, 32, 32, 0)
	mask := WalkMask{Size: 32, Bits: make([]byte, 32*32/8)}
	for z := 0; z < 22; z++ {
		for x := 14; x < 17; x++ {
			bit := z*32 + x
			mask.Bits[bit/8] |= 1 << uint(bit%8)
		}
	}
	nav := baked(t, g, mask)
	path, ok := nav.Path(Point{5, 0, 5}, Point{25, 0, 5})
	if !ok {
		t.Fatal("masked route not found")
	}
	detour := false
	for _, p := range path {
		if p[2] > 22 {
			detour = true
		}
	}
	if !detour {
		t.Fatalf("route ignored server mask: %v", path)
	}
}

func TestFullyBlockedMapFailsClosed(t *testing.T) {
	g := &mapgeometry.Geometry{}
	g.Reindex()
	floor(g, 0, 0, 32, 32, 0)
	mask := WalkMask{Size: 32, Bits: make([]byte, 128)}
	for i := range mask.Bits {
		mask.Bits[i] = 255
	}
	nav := baked(t, g, mask)
	if _, ok := nav.Project(Point{5, 0, 5}, 1, 1); ok {
		t.Fatal("fully blocked map admitted a spawn")
	}
	if _, ok := nav.Path(Point{5, 0, 5}, Point{10, 0, 5}); ok {
		t.Fatal("fully blocked map admitted a route")
	}
}

func TestDenseCrossingsDoNotReduceMovementBudget(t *testing.T) {
	g := &mapgeometry.Geometry{}
	g.Reindex()
	floor(g, 0, 0, 32, 32, 0)
	nav := baked(t, g)
	path := make([]Point, 0, 101)
	for i := 0; i <= 100; i++ {
		path = append(path, Point{5 + float32(i)*.02, 0, 5})
	}
	p, d, consumed, ok := nav.Step(Point{5, 0, 5}, path, 2, func(float32, float32) bool { return true })
	if !ok || abs(p[0]-7) > .001 || abs(d-2) > .001 || consumed != len(path) {
		t.Fatalf("dense path slowed: %v distance=%g consumed=%d/%d", p, d, consumed, len(path))
	}
	// A real corner must still end the packet at the turn, never its diagonal.
	p, d, consumed, ok = nav.Step(Point{5, 0, 5}, []Point{{5, 0, 5}, {5, 0, 7}, {7, 0, 7}}, 4, func(float32, float32) bool { return true })
	if !ok || p != (Point{5, 0, 7}) || abs(d-2) > .001 || consumed != 2 {
		t.Fatalf("cut real corner: %v distance=%g consumed=%d", p, d, consumed)
	}
}

func TestPlayerResourceGenerationAndCorruption(t *testing.T) {
	if os.Getenv("GX_RECAST_BAKER") == "" {
		t.Skip("requires baker")
	}
	g := &mapgeometry.Geometry{}
	g.Reindex()
	floor(g, 0, 0, 20, 20, 0)
	directory := t.TempDir()
	if err := Bake(g, 68, directory); err != nil {
		t.Fatal(err)
	}
	if err := WriteCatalog(directory, []uint16{68}); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateCatalog(directory); err != nil {
		t.Fatal(err)
	}
	nav, err := Load(directory, 68)
	if err != nil {
		t.Fatal(err)
	}
	physics, ok := nav.PlayerGeometry()
	if !ok || physics.Index.Len() != 2 {
		t.Fatal("missing future-map player geometry")
	}
	before, err := os.ReadFile(filepath.Join(directory, "68.gxnav"))
	if err != nil {
		t.Fatal(err)
	}
	if err = WritePlayerPhysics(g, 68, directory); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(directory, "68.gxnav"))
	if !bytes.Equal(before, after) {
		t.Fatal("physics-only update replaced navigation")
	}
	manifestFile := filepath.Join(directory, "68.manifest.json")
	originalManifest, err := os.ReadFile(manifestFile)
	if err != nil {
		t.Fatal(err)
	}
	var obsolete Manifest
	if err = json.Unmarshal(originalManifest, &obsolete); err != nil {
		t.Fatal(err)
	}
	obsolete.CollisionRules = 1
	obsoleteData, _ := json.Marshal(obsolete)
	if err = os.WriteFile(manifestFile, obsoleteData, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Load(directory, 68); err == nil {
		t.Fatal("obsolete construction accepted despite matching file hashes")
	}
	if _, err = ValidateCatalog(directory); err == nil {
		t.Fatal("obsolete construction passed installation validation")
	}
	if err = WritePlayerPhysics(g, 68, directory); err == nil {
		t.Fatal("physics-only update certified an obsolete navigation mesh")
	}
	if err = os.WriteFile(manifestFile, originalManifest, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(directory, "68.gxphysics"), []byte("damaged"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Load(directory, 68); err == nil {
		t.Fatal("damaged player resources accepted")
	}
}
