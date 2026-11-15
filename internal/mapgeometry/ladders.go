package mapgeometry

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"path"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
)

const LadderSpeed = float32(2.5)
const (
	LadderDetach uint8 = iota
	LadderHold
	LadderUp
	LadderDown
)

// Ladder contains derived placement facts, never model or texture data.
type Ladder struct {
	Bottom, Top Position
	Normal      Position
	Width       float32
}

//go:embed native_ladders.json
var ladderCatalog []byte
var nativeLadders = func() map[uint16][]Ladder {
	var v map[uint16][]Ladder
	if err := json.Unmarshal(ladderCatalog, &v); err != nil {
		panic(err)
	}
	return v
}()

func NativeLadders(id uint16) []Ladder { return append([]Ladder(nil), nativeLadders[id]...) }

// BuildLadders reads the original WLD Ladder/Object section and display extents.
// Collision triangles remain unchanged: ladder meshes are climb guides, not walls.
func BuildLadders(read func(string) ([]byte, error), layout *world.WLD) ([]Ladder, error) {
	var result []Ladder
	for _, inst := range layout.Objects.Instances {
		if inst.Asset < 0 || int(inst.Asset) >= len(layout.Objects.Assets) {
			continue
		}
		name := path.Base(strings.ReplaceAll(layout.Objects.Assets[inst.Asset], `\`, "/"))
		raw, err := read("Entity/Object/" + name)
		if err != nil {
			return nil, err
		}
		model, err := world.ParseSMOD(raw)
		if err != nil {
			return nil, err
		}
		low := world.Vec3{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}
		high := world.Vec3{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}
		for _, mesh := range model.Meshes {
			for _, v := range mesh.Vertices {
				for a := range 3 {
					low[a] = min(low[a], v.Pos[a])
					high[a] = max(high[a], v.Pos[a])
				}
			}
		}
		// Native Objects also contains non-climbable trigger boxes.
		if high[1]-low[1] < 2 || high[1]-low[1] < high[0]-low[0] {
			continue
		}
		matrix := Basis(inst.Pos, inst.Fwd, inst.Up)
		bottom := Point(world.Vec3{(low[0] + high[0]) / 2, low[1], (low[2] + high[2]) / 2}, matrix)
		top := Point(world.Vec3{(low[0] + high[0]) / 2, high[1], (low[2] + high[2]) / 2}, matrix)
		normal := norm(world.Vec3{inst.Fwd[0], 0, inst.Fwd[2]})
		if top[1]-bottom[1] < 2 || normal == (world.Vec3{}) {
			continue
		}
		result = append(result, Ladder{Bottom: pv(bottom), Top: pv(top), Normal: Position(normal), Width: max(.4, (high[0]-low[0])/2)})
	}
	return result, nil
}

type LadderMotion struct {
	Active bool
	Index  int
	Side   float32
	Mode   uint8
}

func (g *Geometry) AttachLadder(p, facing Position, index int) (LadderMotion, Position, bool) {
	if index < 0 || index >= len(g.Ladders) {
		return LadderMotion{}, p, false
	}
	l := g.Ladders[index]
	if p[1] < l.Bottom[1]-.8 || p[1] > l.Top[1]+.8 {
		return LadderMotion{}, p, false
	}
	center := l.at(p[1])
	d := cv(p).Sub(cv(center))
	normal := cv(l.Normal)
	depth := d.Dot(normal)
	side := float32(1)
	if depth < 0 {
		side = -1
	}
	lateral := d.Sub(normal.Scale(depth))
	lateral[1] = 0
	if math.Abs(depth) > 1.1 || lateral.Len() > float64(min(l.Width, .9)+.25) {
		return LadderMotion{}, p, false
	}
	// Approach the rungs; walking past a ladder does not capture the character.
	if cv(facing).Dot(normal)*float64(side) > -.25 {
		return LadderMotion{}, p, false
	}
	if g.Supported(p) && p[1] > l.Bottom[1]+1.5 {
		other := center
		for a := range 3 {
			other[a] -= l.Normal[a] * side * .55
		}
		if !g.Supported(other) && g.clearBody(other) {
			side = -side
		}
	}
	q := center
	for a := range 3 {
		q[a] += l.Normal[a] * side * .55
	}
	q[1] = max(l.Bottom[1], min(l.Top[1]+FootSkin, p[1]))
	if !g.clearBody(q) {
		return LadderMotion{}, p, false
	}
	// Do not snap through a wall to reach the guide.
	delta := pv(cv(q).Sub(cv(p)))
	next, _ := g.MoveFeet(p, delta)
	if cv(next).Sub(cv(q)).Len() > .12 {
		return LadderMotion{}, p, false
	}
	return LadderMotion{Active: true, Index: index, Side: side, Mode: LadderHold}, q, true
}
func (l Ladder) at(y float32) Position {
	t := max(float32(0), min(float32(1), (y-l.Bottom[1])/(l.Top[1]-l.Bottom[1])))
	p := l.Bottom
	for a := range 3 {
		p[a] += (l.Top[a] - l.Bottom[a]) * t
	}
	p[1] = y
	return p
}
func (m *LadderMotion) Step(g *Geometry, p Position, dt float32) Position {
	if !m.Active || m.Index < 0 || m.Index >= len(g.Ladders) {
		m.Active = false
		return p
	}
	if m.Mode == LadderHold {
		return p
	}
	l := g.Ladders[m.Index]
	dy := LadderSpeed * max(float32(0), min(dt, .1))
	if m.Mode == LadderDown {
		dy = -dy
	}
	q := l.at(max(l.Bottom[1], min(l.Top[1]+FootSkin, p[1]+dy)))
	for a := range 3 {
		if a != 1 {
			q[a] += l.Normal[a] * m.Side * .55
		}
	}
	q, _ = g.MoveFeet(p, pv(cv(q).Sub(cv(p))))
	if m.Mode == LadderUp && q[1] >= l.Bottom[1]+1.5 {
		if exit, ok := g.ladderExit(q, l, m.Side); ok {
			m.Active = false
			return exit
		}
	}
	if m.Mode == LadderDown && q[1] <= l.Bottom[1]+.06 {
		m.Active = false
	}
	return q
}
func (g *Geometry) ladderExit(p Position, l Ladder, side float32) (Position, bool) {
	// Step onto a real nearby landing; never teleport through a wall or ceiling.
	for _, sign := range []float32{-side, side} {
		for _, distance := range []float32{.55, .85, 1.15, 1.45} {
			q := l.at(p[1])
			q[0] += l.Normal[0] * sign * distance
			q[2] += l.Normal[2] * sign * distance
			if y, ok := g.Floor(q, .2, false); ok {
				q[1] = y + FootSkin
				if !g.clearBody(q) {
					continue
				}
				next, _ := g.MoveFeet(p, pv(cv(q).Sub(cv(p))))
				if cv(next).Sub(cv(q)).Len() < .12 {
					return q, true
				}
			}
		}
	}
	return p, false
}
func ValidateLadders(ladders []Ladder) error {
	for _, l := range ladders {
		if l.Top[1] <= l.Bottom[1] {
			return fmt.Errorf("invalid ladder height")
		}
	}
	return nil
}
