package scene

import (
	"fmt"
	"math"
	"path"
	"strings"
	"sync/atomic"

	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type cloakGeometry struct {
	record   model.CloakRecord
	vertices []render.Vertex
	mesh     *render.Mesh
	shoulder *skinPart
	texture  *render.Texture
	grid     *clothGrid
}
type cloakInstance struct {
	id, version uint64
	geometry    *cloakGeometry
	physics     *clothPhysics
	coarse      *clothPhysics
	coarseLocal []render.V3
	usingGrid   bool
	emblem      *cloakEmblem
	assets      *Assets
	vertices    []render.Vertex
	initialized bool
	prepared    bool
	reused      bool
	min, max    render.V3
	world       render.M4
	elapsed     float32
	lodInterval float64
	lodAt       float64
}

var cloakIDs atomic.Uint64

// HasCloak reports successfully loaded native cloth, including Model zero.
func (c *Character) HasCloak() bool { return c != nil && c.cloak != nil }

func cloakPrefix(a Appearance) string {
	family := "f"
	switch a.Race {
	case 0:
		if a.Job >= 4 {
			family = "m"
		}
	case 1, 2:
		family = "r"
		if a.Job >= 4 {
			family = "m"
		}
	case 3:
		if a.Job >= 2 {
			family = "r"
		}
	}
	sex := "m"
	if a.Gender == 1 {
		sex = "w"
	}
	return racePrefix[a.Race%4] + sex + family
}

// Native 0x414580 selects PC by the chest's ReqVg-1 (0..7), then indexes
// the record and texture by the cloak's zero-based Model byte.
func (s *Assets) cloakGeometry(a Appearance) (*cloakGeometry, error) {
	prefix, folder := cloakPrefix(a), raceFolders[a.Race%4]
	file := fmt.Sprintf("cloak/Character/%s/%d_%s.PC", folder, a.CloakFit+1, prefix)
	key := fmt.Sprintf("%s:%d", file, a.CloakModel)
	if g := s.cloaks[key]; g != nil {
		return g, nil
	}
	raw, err := s.Arc.ReadFile(file)
	if err != nil {
		return nil, err
	}
	table, err := model.ParseCloakPC(raw)
	if err != nil {
		return nil, err
	}
	if int(a.CloakModel) >= len(table.Records) {
		return nil, fmt.Errorf("cloak: model %d outside PC", a.CloakModel)
	}
	record := table.Records[a.CloakModel]
	if record.Mode != 0 && record.Mode != 1 {
		return nil, fmt.Errorf("cloak: unsupported native mode %d", record.Mode)
	}
	raw, err = s.Arc.ReadFile(fmt.Sprintf("cloak/Character/%s/3DC/%s", folder, path.Base(table.Bodies[record.Body])))
	if err != nil {
		return nil, err
	}
	mesh, err := model.ParseCloth3DC(raw)
	if err != nil {
		return nil, err
	}
	if len(mesh.Vertices) != record.Columns*record.Rows {
		return nil, fmt.Errorf("cloak: grid/mesh mismatch")
	}
	short := strings.ToUpper(racePrefix[a.Race%4])
	raw, err = s.Arc.ReadFile("cloak/ClothTexture/CLOTH_TEXTHRE_" + short + ".CTL")
	if err != nil {
		return nil, err
	}
	textures, err := model.ParseClothTextures(raw)
	if err != nil {
		return nil, err
	}
	if record.Texture < 0 || record.Texture >= len(textures) {
		return nil, fmt.Errorf("cloak: invalid texture")
	}
	g := &cloakGeometry{record: record, vertices: make([]render.Vertex, len(mesh.Vertices))}
	for i, v := range mesh.Vertices {
		g.vertices[i] = render.Vertex{Pos: render.V3(v.Pos), Normal: render.V3(v.Normal), UV: v.UV}
	}
	indices := make([]uint32, 0, len(mesh.Faces)*3)
	for _, f := range mesh.Faces {
		indices = append(indices, uint32(f[0]), uint32(f[1]), uint32(f[2]))
	}
	g.shoulder, err = s.skinFile(fmt.Sprintf("cloak/Character/%s/3DC/Static/%s", folder, path.Base(table.Shoulders[record.Shoulder])))
	if err != nil {
		return nil, err
	}
	g.texture = s.modelTexture("cloak/ClothTexture/" + short + "/" + path.Base(textures[record.Texture]))
	if g.texture == nil {
		return nil, fmt.Errorf("cloak: missing texture")
	}
	g.mesh = render.NewAnimatedMesh(len(g.vertices), indices)
	g.mesh.UpdateVertices(g.vertices)
	g.grid = makeClothGrid(record, g.vertices)
	if s.cloaks == nil {
		s.cloaks = map[string]*cloakGeometry{}
	}
	s.cloaks[key] = g
	return g, nil
}
func (c *Character) attachCloak(a Appearance) {
	if a.CloakType != 24 && a.CloakType != 39 {
		return
	}
	g, err := c.assets.cloakGeometry(a)
	if err != nil {
		c.assets.Missing[err.Error()] = true
		return
	}
	cp := *g.shoulder
	// Native 0x415b88/0x5519c0 draws the shoulder as an opaque skinned
	// part. Its texture alpha is not a cutout mask (often below 0.5).
	// 0x415b75 disables lighting for this draw, then restores it.
	// D3DCULL_NONE set at 0x415212 remains active until 0x415bdb,
	// after the shoulder draw. Its winding is mixed; do not cull it.
	cp.tex, cp.cut, cp.unlit, cp.twoSided = g.texture, false, true, true
	c.parts = append(c.parts, &cp)
	rest := make([]render.V3, len(g.vertices))
	for i, v := range g.vertices {
		rest[i] = v.Pos
	}
	p := newClothPhysics(rest, g.record.Columns, g.record.Rows, g.record.Mode)
	for _, pin := range g.record.Pins {
		p.pins[pin.Vertex] = true
	}
	c.cloak = &cloakInstance{id: cloakIDs.Add(1), geometry: g, physics: p, vertices: append([]render.Vertex(nil), g.vertices...)}
	c.cloak.emblem = c.assets.newCloakEmblem(a.CloakEmblem, *g)
	c.cloak.assets = c.assets
}
func (c *cloakInstance) update(world render.M4, finals []render.M4) {
	c.updateGrid(world, finals, 50, false)
}

func (c *cloakInstance) updateGrid(world render.M4, finals []render.M4, budget int, reduced bool) {
	c.reused = false
	c.version++
	p := c.physics
	pins := c.geometry.record.Pins
	grid := c.geometry.grid
	reset := !c.initialized || (render.V3{world[12] - c.world[12], world[13] - c.world[13], world[14] - c.world[14]}).Len() > 20
	if reduced {
		if c.coarse == nil {
			c.coarse = grid.simulation()
			c.coarseLocal = make([]render.V3, len(grid.nodes))
		}
		p, pins = c.coarse, grid.pins
		if !reset && !c.usingGrid {
			for i, node := range grid.nodes {
				p.positions[i], p.velocity[i] = c.physics.positions[node], c.physics.velocity[node]
			}
		}
	} else if !reset && c.usingGrid {
		// Resume from the visible surface and its interpolated velocities.
		for i, sample := range grid.samples {
			p.positions[i] = transformPoint(c.vertices[i].Pos, c.world)
			p.velocity[i] = sample.blend(c.coarse.velocity)
		}
	}
	c.usingGrid = reduced
	if reset {
		for i, v := range p.rest {
			p.positions[i] = transformPoint(v, world)
			p.velocity[i] = render.V3{}
		}
		c.initialized = true
	}
	c.world = world
	for _, pin := range pins {
		if pin.Bone < len(finals) {
			p.positions[pin.Vertex] = transformPoint(render.V3(pin.Offset), finals[pin.Bone].Mul(world))
			p.velocity[pin.Vertex] = render.V3{}
		}
	}
	// 0x584ba0: ceil(frameSeconds*200) steps of 0.0005 seconds.
	steps := min(budget, int(math.Ceil(float64(c.elapsed)*200)))
	c.elapsed = 0
	if c.geometry.record.Mode == 1 {
		steps *= 3
	}
	for range steps {
		p.step(.0005)
	}
	// Cloth is unlit in every pass. Keep authored normals and update the
	// positions and bounds used for depth, clipping and the silhouette.
	inverse := world.Inverse()
	if reduced {
		for i, point := range p.positions {
			c.coarseLocal[i] = transformPoint(point, inverse)
		}
		for i, sample := range grid.samples {
			c.setClothPosition(i, sample.blend(c.coarseLocal).Add(sample.restOffset))
		}
	} else {
		for i, v := range p.positions {
			c.setClothPosition(i, transformPoint(v, inverse))
		}
	}
}

func (c *cloakInstance) setClothPosition(i int, point render.V3) {
	c.vertices[i].Pos = point
	if i == 0 {
		c.min, c.max = point, point
		return
	}
	for axis, value := range point {
		c.min[axis], c.max[axis] = min(c.min[axis], value), max(c.max[axis], value)
	}
}
func (c *cloakInstance) draw(s *render.Scene3D, world render.M4, concealed bool) {
	s.Begin()
	c.drawInPass(s, world, concealed)
}

func (c *cloakInstance) drawInPass(s *render.Scene3D, world render.M4, concealed bool) {
	c.geometry.mesh.UpdateVertices(c.vertices)
	if concealed {
		s.DrawCloth(c.geometry.mesh, c.geometry.texture, world, .5, render.StealthOpacity)
	} else {
		s.DrawCloth(c.geometry.mesh, c.geometry.texture, world, .5, 1)
	}
	c.drawEmblem(s, c.assets, world, concealed)
}
