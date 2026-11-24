package scene

import (
	"math"
	"path"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/clientlog"
	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Phoenix WATER_SURFACE_Y and the original water controllers share raw Y=0.
const waterSurfaceY = float32(0)

const SwimEntryY = waterSurfaceY - 1.6
const SwimSurfaceY = waterSurfaceY - 1.5

// WaterFloor reports terrain deep enough for the map's water to support swimming.
func (w *World) WaterFloor(x, z float32) (float32, bool) {
	if w == nil || w.water == nil || !w.HasTerrain() || x < 0 || z < 0 || x > float32(w.WLD.MapSize) || z > float32(w.WLD.MapSize) {
		return 0, false
	}
	floor := w.Height(x, z)
	return floor, floor < SwimEntryY
}

// SwimmingAt uses the authored map-wide WTR surface and the original
// character thresholds documented in Phoenix (before its +0.25 adjustment).
// It is presentation state; positions remain server-confirmed.
func (w *World) SwimmingAt(x, y, z float32) bool {
	_, deep := w.WaterFloor(x, z)
	return deep && y <= SwimSurfaceY
}

// WaterContactAt excludes dry ground, falling actors and deep divers. Surface
// swimmers and grounded waders leave rings at the authored WTR surface.
func (w *World) WaterContactAt(pos render.V3) bool {
	if w == nil || w.water == nil || !w.HasTerrain() || pos[0] < 0 || pos[2] < 0 || pos[0] > float32(w.WLD.MapSize) || pos[2] > float32(w.WLD.MapSize) {
		return false
	}
	floor := w.Height(pos[0], pos[2])
	return floor < 0 && pos[1] < .1 && pos[1] >= SwimEntryY-.18 && pos[1] >= floor-.1
}

// UnderwaterFogAt follows the camera, including in shallow water. The fog
// colour blends Phoenix's underwater reference with the map-authored WTR tint;
// the precise original-client underwater shader remains unverified.
func (w *World) UnderwaterFogAt(eye render.V3) (render.V3, bool) {
	if w == nil || w.water == nil || !w.HasTerrain() || eye[0] < 0 || eye[2] < 0 || eye[0] > float32(w.WLD.MapSize) || eye[2] > float32(w.WLD.MapSize) || eye[1] >= waterSurfaceY || eye[1] < w.Height(eye[0], eye[2]) {
		return render.V3{}, false
	}
	t := w.water.tint
	return render.V3{.8*.128 + .2*t[0], .8*.295 + .2*t[1], .8*.382 + .2*t[2]}, true
}

type waterSurface struct {
	mesh       *render.Mesh
	frames     []*render.Texture
	definition *world.WTR
	tint       [4]float32
	normals    [2]*render.Texture
	wetCells   []cellKey
	indexed    bool
}

func loadWater(a *Assets, w *World) *waterSurface {
	if !w.HasTerrain() || !strings.EqualFold(path.Ext(w.WLD.Layout), ".wtr") {
		return nil
	}
	size := float32(w.WLD.MapSize)
	water := &waterSurface{tint: [4]float32{.12, .32, .42, 1}, mesh: render.NewMesh([]render.Vertex{
		{Pos: render.V3{0, waterSurfaceY, 0}, UV2: [2]float32{0, 0}},
		{Pos: render.V3{size, waterSurfaceY, 0}, UV2: [2]float32{size, 0}},
		{Pos: render.V3{size, waterSurfaceY, size}, UV2: [2]float32{size, -size}},
		{Pos: render.V3{0, waterSurfaceY, size}, UV2: [2]float32{0, -size}},
	}, []uint32{0, 1, 2, 0, 2, 3})}
	water.indexTerrain(w)
	file := "Entity/Water/" + path.Base(strings.ReplaceAll(w.WLD.Layout, `\`, "/"))
	b, err := a.Arc.ReadFile(file)
	if err == nil {
		water.definition, err = world.ParseWTR(b)
	}
	if err != nil {
		clientlog.Warnf("water %s: %v", file, err)
		return water
	}
	d := water.definition
	water.tint = [4]float32{float32(d.ARGB>>16&255) / 255, float32(d.ARGB>>8&255) / 255, float32(d.ARGB&255) / 255, float32(d.ARGB>>24) / 255}
	for i, name := range []string{"WaterBumps1.tga", "WaterBumps2.tga"} {
		file := "Entity/Water/" + name
		if _, ok := a.Arc.Lookup(file); !ok {
			file = name // Legacy loose packs store the height maps at the data root.
		}
		if data, err := a.Arc.ReadFile(file); err == nil {
			if height, err := img.DecodeTGA(data); err == nil {
				if normal := img.WaterNormals(height); normal != nil {
					water.normals[i] = render.Upload(img.Texture{RGBA: normal})
					water.normals[i].SetWrap(true)
				}
			}
		}
	}
	for _, name := range d.Textures {
		name = "Entity/Water/" + DDSName(path.Base(strings.ReplaceAll(name, `\`, "/")))
		var t *render.Texture
		b, err := a.Arc.ReadFile(name)
		if err == nil {
			var decoded img.Texture
			decoded, err = img.Decode(b)
			if err == nil {
				t = render.Upload(decoded)
				t.SetWrap(true)
			}
		}
		if err != nil {
			clientlog.Warnf("water frame %s: %v", name, err)
		}
		water.frames = append(water.frames, t)
	}
	return water
}

// A triangle can meet the plane only if at least one of its vertices is at or
// below it. Index whole cells (including shared edge vertices) conservatively.
func (water *waterSurface) indexTerrain(w *World) {
	water.wetCells, water.indexed = nil, false
	if w.stream == nil || w.side < 2 || len(w.heights) < w.side*w.side {
		return
	}
	for z := 0; z < w.side-1; z += terrainCellSamples {
		for x := 0; x < w.side-1; x += terrainCellSamples {
			wet := false
			for iz := z; iz <= min(z+terrainCellSamples, w.side-1) && !wet; iz++ {
				for ix := x; ix <= min(x+terrainCellSamples, w.side-1); ix++ {
					if w.heights[iz*w.side+ix] <= waterSurfaceY {
						wet = true
						break
					}
				}
			}
			if wet {
				water.wetCells = append(water.wetCells, cellKey{x / terrainCellSamples, z / terrainCellSamples})
			}
		}
	}
	water.indexed = true
}

// DrawWater follows the original WTR flipbook. Transparent surfaces draw after
// opaque actors/geometry, so submerged models retain the water's attenuation.
func (w *World) DrawWater(s *render.Scene3D) {
	if w.WaterVisible(s.View, s.Proj) {
		w.DrawVisibleWater(s)
	}
}

// DrawVisibleWater accepts the visibility result from the same camera pass.
func (w *World) DrawVisibleWater(s *render.Scene3D) {
	water := w.water
	s.Begin()
	if len(water.frames) > 0 {
		index := water.definition.FrameAt(w.elapsed)
		if frame := water.frames[index]; frame != nil {
			s.WaterNormals = water.normals
			// All four authored scroll rates wrap together after 200/3 units.
			s.WaterTime = float32(math.Mod(w.elapsed*1.5, 200.0/3))
			s.DrawWater(water.mesh, frame, water.definition.TileSize, water.tint)
			s.WaterNormals = [2]*render.Texture{}
			return
		}
	}
	// Keep the previous simple surface as a visible fallback for missing assets.
	s.DrawSolid(water.mesh, [4]float32{.12, .32, .42, 1})
}

// WaterVisible is conservative for the map-wide flat WTR surface.
func (w *World) WaterVisible(view, projection render.M4) bool {
	visible, _ := w.WaterCoverage(view, projection, false)
	return visible
}

// WaterCoverage bounds the screen region that can sample the reflection.
// An uncertain receiver or near-plane crossing retains the full target.
func (w *World) WaterCoverage(view, projection render.M4, region bool) (bool, render.ScreenRegion) {
	full := render.FullScreenRegion
	if w == nil || w.water == nil || w.WLD == nil {
		return false, full
	}
	size := float32(w.WLD.MapSize)
	frustum := render.ViewFrustum(view, projection)
	if !frustum.BoxVisible(render.V3{0, waterSurfaceY, 0}, render.V3{size, waterSurfaceY, size}) {
		return false, full
	}
	water, stream := w.water, w.stream
	if !water.indexed || stream == nil || !w.Ready() || projection[0] == 0 || projection[5] == 0 || projection[10] <= 1 || projection[14] >= 0 {
		return true, full
	}
	for _, v := range view {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return true, full
		}
	}
	// Inverse() returns identity for a singular matrix; it must not turn an
	// unknown camera into an apparent in-map viewpoint.
	det := view[0]*(view[5]*view[10]-view[6]*view[9]) - view[4]*(view[1]*view[10]-view[2]*view[9]) + view[8]*(view[1]*view[6]-view[2]*view[5])
	if math.Abs(float64(det)) < 1e-6 || view[3] != 0 || view[7] != 0 || view[11] != 0 || view[15] != 1 {
		return true, full
	}
	far := projection[14] / (1 - projection[10])
	near := -projection[14] / projection[10]
	if far <= 0 || near <= 0 || math.IsNaN(float64(far)) || math.IsInf(float64(far), 0) {
		return true, full
	}
	// Only trust occlusion when the full view is resident and all loaded
	// terrain sections have at least one opaque layer.
	rx, ry := float64(1/projection[0]), float64(1/projection[5])
	required := far * float32(math.Sqrt(1+rx*rx+ry*ry))
	if math.IsNaN(float64(required)) || math.IsInf(float64(required), 0) || stream.radius+stream.size()*.25 < required {
		return true, full
	}
	for _, mat := range stream.materials {
		opaque := false
		for _, layer := range mat.Layers {
			if layer.Mask != nil && layer.Tile != nil {
				opaque = true
				break
			}
		}
		if !opaque {
			return true, full
		}
	}
	if len(stream.materials) == 0 {
		return true, full
	}
	inverse := view.Inverse()
	eye := render.V3{inverse[12], inverse[13], inverse[14]}
	if math.IsNaN(float64(eye[0])) || math.IsNaN(float64(eye[1])) || math.IsNaN(float64(eye[2])) || math.IsInf(float64(eye[0]), 0) || math.IsInf(float64(eye[1]), 0) || math.IsInf(float64(eye[2]), 0) || eye[0] < 0 || eye[2] < 0 || eye[0] > size || eye[2] > size || eye[1] <= w.Height(eye[0], eye[2])+near*2 {
		return true, full
	}
	cellSize := stream.size()
	if stream.center != (cellKey{int(math.Floor(float64(eye[0] / cellSize))), int(math.Floor(float64(eye[2] / cellSize)))}) {
		return true, full
	}
	rect := render.ScreenRegion{X0: 1, Y0: 1}
	minDepth := float32(math.MaxFloat32)
	vp := view.Mul(projection)
	visible := false
	for _, cell := range water.wetCells {
		x, z := float32(cell.x)*cellSize, float32(cell.z)*cellSize
		lo := render.V3{x - .01, waterSurfaceY - .01, z - .01}
		hi := render.V3{min(x+cellSize, size) + .01, waterSurfaceY + .01, min(z+cellSize, size) + .01}
		if !frustum.BoxVisible(lo, hi) {
			continue
		}
		visible = true
		if !region {
			return true, full
		}
		for i := range 8 {
			p := lo
			for axis := range 3 {
				if i&(1<<axis) != 0 {
					p[axis] = hi[axis]
				}
			}
			depth := p[0]*vp[3] + p[1]*vp[7] + p[2]*vp[11] + vp[15]
			if depth <= near {
				return true, full
			}
			minDepth = min(minDepth, depth)
			u := ((p[0]*vp[0]+p[1]*vp[4]+p[2]*vp[8]+vp[12])/depth + 1) * .5
			v := ((p[0]*vp[1]+p[1]*vp[5]+p[2]*vp[9]+vp[13])/depth + 1) * .5
			rect.X0, rect.X1 = min(rect.X0, u), max(rect.X1, u)
			rect.Y0, rect.Y1 = min(rect.Y0, v), max(rect.Y1, v)
		}
	}
	if !visible {
		return false, full
	}
	// The four normal samples can shift reflection UV by at most .128*.95/z.
	// Reflections without normal maps use screen UV directly.
	margin := float32(0)
	if w.HasWaterNormals() {
		margin = .122 / max(minDepth, .1)
	}
	return true, rect.Expand(margin, margin)
}

func (w *World) HasWaterNormals() bool {
	return w != nil && w.water != nil && w.water.normals[0] != nil && w.water.normals[1] != nil
}

func (w *waterSurface) Delete() {
	w.mesh.Delete()
	for _, t := range w.normals {
		if t != nil {
			t.Delete()
		}
	}
	for _, t := range w.frames {
		if t != nil {
			t.Delete()
		}
	}
}
