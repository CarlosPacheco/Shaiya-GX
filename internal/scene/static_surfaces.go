package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
)

type staticPlane [4]int64
type staticFace struct {
	points [3]render.V3
	normal render.V3
	area   float32
}

func insideStaticFace(p render.V3, f staticFace) bool {
	for i := range 3 {
		a, b := f.points[i], f.points[(i+1)%3]
		if b.Sub(a).Cross(p.Sub(a)).Dot(f.normal) < -f.area*1e-5 {
			return false
		}
	}
	return true
}

// Detect explicitly authored front/back sheets, including opposite quads with
// different diagonals. Seven samples require coverage by the opposite sheet;
// a merely parallel nearby wall or intersecting triangle is not sufficient.
func coincidentStaticFaces(mesh world.SMODMesh) []bool {
	faces := make([]staticFace, len(mesh.Faces))
	groups := map[staticPlane][]int{}
	for index, face := range mesh.Faces {
		if int(face[0]) >= len(mesh.Vertices) || int(face[1]) >= len(mesh.Vertices) || int(face[2]) >= len(mesh.Vertices) {
			continue
		}
		f := staticFace{}
		for i, v := range face {
			f.points[i] = render.V3(mesh.Vertices[v].Pos)
		}
		n := f.points[1].Sub(f.points[0]).Cross(f.points[2].Sub(f.points[0]))
		f.area = n.Len()
		if f.area < 1e-8 {
			continue
		}
		f.normal = n.Scale(1 / f.area)
		faces[index] = f
		canonical := f.normal
		axis := 0
		for i := 1; i < 3; i++ {
			if math.Abs(float64(canonical[i])) > math.Abs(float64(canonical[axis])) {
				axis = i
			}
		}
		if canonical[axis] < 0 {
			canonical = canonical.Scale(-1)
		}
		key := staticPlane{}
		for i := range 3 {
			key[i] = int64(math.Round(float64(canonical[i]) * 10000))
		}
		key[3] = int64(math.Round(float64(canonical.Dot(f.points[0])) * 10000))
		groups[key] = append(groups[key], index)
	}
	paired := make([]bool, len(faces))
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		opposed := false
		for _, index := range group[1:] {
			if faces[group[0]].normal.Dot(faces[index].normal) < -.9999 {
				opposed = true
				break
			}
		}
		if !opposed {
			continue
		}
		for _, index := range group {
			f := faces[index]
			samples := [7]render.V3{f.points[0], f.points[1], f.points[2], f.points[0].Add(f.points[1]).Scale(.5), f.points[1].Add(f.points[2]).Scale(.5), f.points[2].Add(f.points[0]).Scale(.5), f.points[0].Add(f.points[1]).Add(f.points[2]).Scale(1.0 / 3)}
			all := true
			for _, p := range samples {
				covered := false
				for _, other := range group {
					g := faces[other]
					if f.normal.Dot(g.normal) > -.9999 || math.Abs(float64(p.Sub(g.points[0]).Dot(g.normal))) > 1e-5 {
						continue
					}
					if insideStaticFace(p, g) {
						covered = true
						break
					}
				}
				if !covered {
					all = false
					break
				}
			}
			paired[index] = all
		}
	}
	return paired
}

// Compact split parts so a small set of paired faces does not duplicate the
// entire source vertex buffer or change model-wide bounds with unused vertices.
func staticSurfacePart(source world.SMODMesh, faces []world.Face) world.SMODMesh {
	if len(faces) == len(source.Faces) {
		return source
	}
	result := world.SMODMesh{Texture: source.Texture}
	remap := map[uint16]uint16{}
	for _, face := range faces {
		rewritten := world.Face{}
		if int(face[0]) >= len(source.Vertices) || int(face[1]) >= len(source.Vertices) || int(face[2]) >= len(source.Vertices) {
			continue
		}
		for i, old := range face {
			next, ok := remap[old]
			if !ok {
				next = uint16(len(result.Vertices))
				remap[old] = next
				result.Vertices = append(result.Vertices, source.Vertices[old])
			}
			rewritten[i] = next
		}
		result.Faces = append(result.Faces, rewritten)
	}
	return result
}

// Keep unpaired cutout sheets two-sided. Only genuinely coincident front/back
// surfaces use clockwise back-face culling; vertices, UVs and collisions survive.
func prepareStaticSurfaces(source *world.SMOD) (*world.SMOD, []bool) {
	result := *source
	result.Meshes = nil
	culled := []bool{}
	for _, mesh := range source.Meshes {
		flags := coincidentStaticFaces(mesh)
		paired, unpaired := []world.Face{}, []world.Face{}
		for i, face := range mesh.Faces {
			if flags[i] {
				paired = append(paired, face)
			} else {
				unpaired = append(unpaired, face)
			}
		}
		if len(unpaired) > 0 || len(mesh.Faces) == 0 {
			part := staticSurfacePart(mesh, unpaired)
			result.Meshes = append(result.Meshes, part)
			culled = append(culled, false)
		}
		if len(paired) > 0 {
			part := staticSurfacePart(mesh, paired)
			result.Meshes = append(result.Meshes, part)
			culled = append(culled, true)
		}
	}
	return &result, culled
}
