// Package mapgeometry builds the shared, graphics-free original world geometry.
package mapgeometry

import (
	"crypto/sha256"
	"fmt"
	"math"
	"path"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
)

type Matrix [16]float32
type Dynamic struct {
	bounds *world.Box
	Meshes []world.CollisionMesh
	Base   Matrix
	Axis   world.Vec3
	Speed  float32
}
type Geometry struct {
	Ladders   []Ladder
	clock     func() float64
	Water     bool
	Triangles []collision.Triangle
	Dynamic   []Dynamic
	MapSize   uint32
	Heights   []uint16
	Sources   map[string]string
	Index     *collision.Index
}

func Identity() Matrix { return Matrix{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1} }
func norm(v world.Vec3) world.Vec3 {
	length := float32(math.Sqrt(float64(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])))
	if length == 0 {
		return v
	}
	for i := range v {
		v[i] *= 1 / length
	}
	return v
}
func cross(a, b world.Vec3) world.Vec3 {
	return world.Vec3{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}
func Basis(p, f, u world.Vec3) Matrix {
	f, u = norm(f), norm(u)
	r := norm(cross(u, f))
	return Matrix{r[0], r[1], r[2], 0, u[0], u[1], u[2], 0, f[0], f[1], f[2], 0, p[0], p[1], p[2], 1}
}
func Point(p world.Vec3, m Matrix) collision.Vec {
	return collision.Vec{float64(p[0]*m[0] + p[1]*m[4] + p[2]*m[8] + m[12]), float64(p[0]*m[1] + p[1]*m[5] + p[2]*m[9] + m[13]), float64(p[0]*m[2] + p[1]*m[6] + p[2]*m[10] + m[14])}
}
func Visit(meshes []world.CollisionMesh, m Matrix, emit func(collision.Triangle)) {
	for _, mesh := range meshes {
		for _, f := range mesh.Faces {
			if int(f[0]) >= len(mesh.Vertices) || int(f[1]) >= len(mesh.Vertices) || int(f[2]) >= len(mesh.Vertices) {
				continue
			}
			t := collision.Triangle{Point(mesh.Vertices[f[0]], m), Point(mesh.Vertices[f[1]], m), Point(mesh.Vertices[f[2]], m)}
			if t.Valid() {
				emit(t)
			}
		}
	}
}
func (g *Geometry) Add(t collision.Triangle) {
	if t.Valid() {
		g.Triangles = append(g.Triangles, t)
		g.Index.Add(t)
	}
}
func (g *Geometry) Reindex() {
	g.Index = collision.NewIndex()
	for _, t := range g.Triangles {
		g.Index.Add(t)
	}
}
func Load(readFile func(string) ([]byte, error), layout *world.WLD) (*Geometry, error) {
	g := &Geometry{Water: !layout.Dungeon && strings.EqualFold(path.Ext(layout.Layout), ".wtr"), Index: collision.NewIndex(), MapSize: layout.MapSize, Heights: layout.Heights, Sources: map[string]string{}}
	read := func(file string) ([]byte, error) {
		file = strings.ToLower(strings.ReplaceAll(file, `\`, "/"))
		data, err := readFile(file)
		if err == nil {
			g.Sources[file] = fmt.Sprintf("%x", sha256.Sum256(data))
		}
		return data, err
	}
	cache := map[string][]world.CollisionMesh{}
	mesh := func(file string) ([]world.CollisionMesh, error) {
		file = strings.ToLower(strings.ReplaceAll(file, `\`, "/"))
		if m, ok := cache[file]; ok {
			return m, nil
		}
		data, err := read(file)
		if err != nil {
			return nil, err
		}
		var m []world.CollisionMesh
		switch strings.ToLower(path.Ext(file)) {
		case ".dg":
			m, err = world.ParseDGCollisions(data)
		case ".vani":
			m, err = world.ParseVANICollisions(data)
		default:
			if strings.HasPrefix(file, "entity/tree/") {
				m, err = world.ParseTreeCollisions(data)
			} else {
				m, err = world.ParseSMODCollisions(data)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("collision %s: %w", file, err)
		}
		for _, part := range m {
			for _, p := range part.Vertices {
				if !Point(p, Identity()).Finite() {
					return nil, fmt.Errorf("collision %s: non-finite vertex", file)
				}
			}
			for _, f := range part.Faces {
				for _, id := range f {
					if int(id) >= len(part.Vertices) {
						return nil, fmt.Errorf("collision %s: invalid face index", file)
					}
				}
			}
		}
		cache[file] = m
		return m, nil
	}
	for _, entry := range []struct {
		group world.Group
		dir   string
	}{{layout.Buildings, "Entity/Building"}, {layout.Shapes, "Entity/Shape"}, {layout.Trees, "Entity/Tree"}, {layout.VaniA, "Entity/VAni"}, {layout.Dungeons, "world/dungeon"}} {
		for _, instance := range entry.group.Instances {
			if instance.Asset < 0 || int(instance.Asset) >= len(entry.group.Assets) {
				continue
			}
			name := path.Base(strings.ReplaceAll(entry.group.Assets[instance.Asset], `\`, "/"))
			if entry.dir == "world/dungeon" && !strings.EqualFold(path.Ext(name), ".dg") {
				continue
			}
			m, err := mesh(entry.dir + "/" + name)
			if err != nil {
				return nil, err
			}
			Visit(m, Basis(instance.Pos, instance.Fwd, instance.Up), g.Add)
		}
	}
	for _, instance := range layout.Manis {
		if instance.Building < 0 || int(instance.Building) >= len(layout.Buildings.Assets) {
			continue
		}
		m, err := mesh("Entity/Building/" + layout.Buildings.Assets[instance.Building])
		if err != nil {
			return nil, err
		}
		base := Basis(instance.Pos, instance.Fwd, instance.Up)
		var animation world.Mani
		if instance.Mani >= 0 && int(instance.Mani) < len(layout.ManiAssets) {
			if data, err := read("Entity/Mani/" + path.Base(strings.ReplaceAll(layout.ManiAssets[instance.Mani], `\`, "/"))); err == nil {
				animation, _ = world.ParseMani(data)
			}
		}
		if animation.Rotate && animation.Speed != 0 && norm(animation.Axis) != (world.Vec3{}) {
			dynamic := Dynamic{Meshes: m, Base: base, Axis: animation.Axis, Speed: animation.Speed}
			dynamic.prepareBounds()
			g.Dynamic = append(g.Dynamic, dynamic)
		} else {
			Visit(m, base, g.Add)
		}
	}
	if layout.Dungeon {
		m, err := mesh("world/dungeon/" + layout.Layout)
		if err != nil {
			return nil, err
		}
		Visit(m, Identity(), g.Add)
	}
	for _, b := range layout.Blockers {
		if !Point(b.Bounds.Min, Identity()).Finite() || !Point(b.Bounds.Max, Identity()).Finite() || math.IsNaN(float64(b.Yaw)) || math.IsInf(float64(b.Yaw), 0) {
			return nil, fmt.Errorf("non-finite WLD blocker")
		}
		for _, t := range collision.BlockerBox(Point(b.Bounds.Min, Identity()), Point(b.Bounds.Max, Identity()), float64(b.Yaw)) {
			g.Add(t)
		}
	}
	return g, nil
}
