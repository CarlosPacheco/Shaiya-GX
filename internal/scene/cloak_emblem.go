package scene

import (
	"path"

	"github.com/Spelunkern/Shaiya-GX/internal/data/emblem"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Cloak emblems (Game.exe 0x414580 builds the object, 0x55a4d0 draws it). The
// six gem bytes of the cloak pick up to three textures. They are drawn over a
// 3x3 patch of the cloth, back layer first, each multiplied by its palette
// colour: bytes 0 front, 1 back, 2 emblem, then their colours.
type emblemTables struct {
	front, back, list []string
	loaded            bool
}

type emblemLayer struct {
	tex  *render.Texture
	tint [4]float32
}

type cloakEmblem struct {
	layers [3]emblemLayer
	nodes  [9]int
}

func (s *Assets) emblemNames() *emblemTables {
	if s.emblems != nil {
		return s.emblems
	}
	t := &emblemTables{}
	s.emblems = t
	load := func(file string) []string {
		raw, err := s.Arc.ReadFile("cloak/Emblem/" + file)
		if err != nil {
			s.Missing["cloak/Emblem/"+file] = true
			return nil
		}
		names, err := emblem.ParseNames(raw)
		if err != nil {
			s.Missing["cloak/Emblem/"+file] = true
			return nil
		}
		return names
	}
	t.list, t.back, t.front = load("EmblemList.dat"), load("EmblemBack.dat"), load("EmblemFront.dat")
	t.loaded = true
	return t
}

// EmblemCount is the number of patterns of a table (0 front, 1 back, 2 emblem).
func (s *Assets) EmblemCount(kind int) int {
	if s == nil || s.Arc == nil {
		return 0
	}
	t := s.emblemNames()
	switch kind {
	case 0:
		return len(t.front)
	case 1:
		return len(t.back)
	}
	return len(t.list)
}

// EmblemTexture loads one pattern texture. kind 0 is the front shape, 1 the
// back shape and 2 the emblem; index is the stored one-based byte.
func (s *Assets) EmblemTexture(kind int, index uint8) *render.Texture {
	if index == 0 || s.Arc == nil {
		return nil
	}
	t := s.emblemNames()
	names, dir := t.front, "BackgroundTex"
	switch kind {
	case 1:
		names = t.back
	case 2:
		names, dir = t.list, "EmblemTex"
	}
	if int(index) > len(names) || names[index-1] == "" {
		return nil
	}
	tex := s.Texture(path.Join("cloak/Emblem", dir, names[index-1]))
	if tex != nil {
		tex.SetWrap(false)
	}
	return tex
}

func emblemTint(color uint8) [4]float32 {
	c := emblem.Color(color)
	return [4]float32{float32(c[0]) / 255, float32(c[1]) / 255, float32(c[2]) / 255, 1}
}

func (s *Assets) newCloakEmblem(pattern [6]uint8, record cloakGeometry) *cloakEmblem {
	p := emblem.FromBytes(pattern)
	if p.Empty() {
		return nil
	}
	e := &cloakEmblem{}
	// Draw order of 0x55a4d0: object +8 (back) with the colour at +0x80, +0xc
	// (front) with +0x84, then +0x10 (emblem) with +0x88.
	e.layers[0] = emblemLayer{s.EmblemTexture(1, p.Back), emblemTint(p.BackColor)}
	e.layers[1] = emblemLayer{s.EmblemTexture(0, p.Front), emblemTint(p.FrontColor)}
	e.layers[2] = emblemLayer{s.EmblemTexture(2, p.Emblem), emblemTint(p.EmblemColor)}
	col, row, dc, dr := record.record.Emblem[0], record.record.Emblem[1], record.record.Emblem[2], record.record.Emblem[3]
	columns := record.record.Columns
	// Vertex order of the original 3x3 buffer: its nine cloth samples.
	at := func(r, c int) int { return min(max(r, 0), record.record.Rows-1)*columns + min(max(c, 0), columns-1) }
	e.nodes = [9]int{
		at(row-dr, col-dc), at(row-dr, col), at(row-dr, col+dc),
		at(row, col-dc), at(row, col), at(row, col+dc),
		at(row+dr, col-dc), at(row+dr, col), at(row+dr, col+dc),
	}
	return e
}

var emblemUV = [9][2]float32{{0, 0}, {.5, 0}, {1, 0}, {0, .5}, {.5, .5}, {1, .5}, {0, 1}, {.5, 1}, {1, 1}}

// emblemIndices are the eight triangles of the native index buffer.
var emblemIndices = []uint32{0, 4, 3, 0, 1, 4, 1, 5, 4, 1, 2, 5, 3, 7, 6, 3, 4, 7, 4, 8, 7, 4, 5, 8}

// emblemLift keeps the layers in front of the cloth they follow, so the decal
// wins the depth test against the cloth it is drawn on. Native 0x55a4d0 draws
// the nine cloth points as they are; GL needs a little room.
const emblemLift = 0.004

func (c *cloakInstance) drawEmblem(s *render.Scene3D, assets *Assets, world render.M4, concealed bool) {
	e := c.emblem
	if e == nil || assets == nil {
		return
	}
	if assets.emblemMesh == nil {
		assets.emblemMesh = render.NewAnimatedMesh(9, emblemIndices)
	}
	var vertices [9]render.Vertex
	for i, node := range e.nodes {
		vertices[i] = render.Vertex{Pos: c.vertices[node].Pos, UV: emblemUV[i]}
	}
	// The cloth normals only serve lighting, so lift along the patch's own
	// normal, on the side facing away from the body (+Z in cloak space).
	n := vertices[5].Pos.Sub(vertices[3].Pos).Cross(vertices[7].Pos.Sub(vertices[1].Pos))
	if n[2] < 0 {
		n = n.Scale(-1)
	}
	if l := n.Len(); l > 0 {
		lift := n.Scale(emblemLift / l)
		for i := range vertices {
			vertices[i].Pos = vertices[i].Pos.Add(lift)
			vertices[i].Normal = n.Scale(1 / l)
		}
	}
	assets.emblemMesh.UpdateVertices(vertices[:])
	for _, layer := range e.layers {
		if layer.tex == nil {
			continue
		}
		tint := layer.tint
		if concealed {
			tint[3] *= render.StealthOpacity
		}
		s.DrawClothOverlay(assets.emblemMesh, layer.tex, world, tint)
	}
}

// SetCloakEmblem swaps the pattern of the worn cloak without rebuilding the
// character, so the cloth keeps its motion while a window edits it.
func (c *Character) SetCloakEmblem(pattern [6]uint8) {
	if c == nil || c.cloak == nil {
		return
	}
	c.cloak.emblem = c.assets.newCloakEmblem(pattern, *c.cloak.geometry)
}
