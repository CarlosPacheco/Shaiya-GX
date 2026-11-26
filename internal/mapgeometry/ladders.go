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

// LadderSpeed is the climbing speed of a character with the given move index:
// the original table 0.8, 1.3, 1.8, 2.3, 2.8 (Game.exe 0x43aea0 and the
// climbing branch of 0x419f60). Other indices use the normal tier.
func LadderSpeed(tier uint8) float32 {
	if tier <= 4 {
		return 0.8 + 0.5*float32(tier)
	}
	return 1.8
}

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
	// Tier is the character's move index; it selects the climbing speed.
	// Speed overrides it when nonzero (geometry fixtures).
	Tier  uint8
	Speed float32
}

func (g *Geometry) AttachLadder(p, facing Position, index int, mode uint8) (LadderMotion, Position, bool) {
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
	// Game.exe 0x443893/0x443977 faces against the authored forward
	// vector at both ends. An upper landing can reach that face to descend.
	if side < 0 && (mode != LadderDown || p[1] < l.Top[1]-1.5 || !g.Supported(p)) {
		return LadderMotion{}, p, false
	}
	q := center
	for a := range 3 {
		q[a] += l.Normal[a] * .55
	}
	q[1] = max(l.Bottom[1], min(l.Top[1]+.55, p[1]))
	if !g.clearBody(q) {
		return LadderMotion{}, p, false
	}
	// Do not snap through a wall to reach the guide.
	delta := pv(cv(q).Sub(cv(p)))
	next, _ := g.MoveFeet(p, delta)
	if cv(next).Sub(cv(q)).Len() > .12 {
		return LadderMotion{}, p, false
	}
	return LadderMotion{Active: true, Index: index, Side: 1, Mode: LadderHold}, q, true
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
	offset := max(float32(.55), min(float32(1.45), float32(cv(p).Sub(cv(l.at(p[1]))).Dot(cv(l.Normal)))*m.Side))
	speed := m.Speed
	if speed <= 0 {
		speed = LadderSpeed(m.Tier)
	}
	dy := speed * max(float32(0), min(dt, .1))
	if m.Mode == LadderDown {
		dy = -dy
	}
	q := l.at(max(l.Bottom[1], min(l.Top[1]+.55, p[1]+dy)))
	for a := range 3 {
		if a != 1 {
			q[a] += l.Normal[a] * m.Side * offset
		}
	}
	wantedY := q[1]
	q, _ = g.MoveFeet(p, pv(cv(q).Sub(cv(p))))
	if m.Mode == LadderUp && q[1] < wantedY-.002 && offset < 1.45 {
		// Clear a platform lip by moving out along the rung normal, never through it.
		extra := min(speed*dt, 1.45-offset)
		outward := Position{l.Normal[0] * m.Side * extra, 0, l.Normal[2] * m.Side * extra}
		shifted, _ := g.MoveFeet(q, outward)
		gained := float32(cv(shifted).Sub(cv(q)).Dot(cv(l.Normal))) * m.Side
		if gained > 0 {
			q = shifted
		}
	}
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
		for _, distance := range []float32{0, .25, .55, .85, 1.15, 1.45} {
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

// LadderVisualAnchor is where a climbing character's model is drawn. The
// original keeps the character's root exactly on the guide line between the
// two derived endpoints (Game.exe 0x537200 builds them from the model's
// centre X, minimum Z and Y extent; 0x43aea0 and 0x43bfa0 never add a
// sideways offset). GX keeps a clearance offset for collision, so only the
// drawn position is pulled onto the guide. Past the top the character is
// stepping onto the landing, so the pull fades out over 0.55 units.
func (g *Geometry) LadderVisualAnchor(index int, p Position) Position {
	if index < 0 || index >= len(g.Ladders) {
		return p
	}
	return g.Ladders[index].visualAnchor(p)
}

// NearestLadderVisualAnchor finds the guide a remote climber is on. It
// returns false when no guide plausibly holds the position.
func (g *Geometry) NearestLadderVisualAnchor(p Position) (Position, bool) {
	best, bestDepth := -1, float32(math.MaxFloat32)
	for i, l := range g.Ladders {
		if p[1] < l.Bottom[1]-.8 || p[1] > l.Top[1]+.8 {
			continue
		}
		d := cv(p).Sub(cv(l.at(p[1])))
		normal := cv(l.Normal)
		depth := d.Dot(normal)
		lateral := d.Sub(normal.Scale(depth))
		lateral[1] = 0
		if math.Abs(depth) > 1.6 || lateral.Len() > float64(min(l.Width, .9)+.3) {
			continue
		}
		if a := float32(math.Abs(depth)); a < bestDepth {
			best, bestDepth = i, a
		}
	}
	if best < 0 {
		return p, false
	}
	return g.Ladders[best].visualAnchor(p), true
}

func (l Ladder) visualAnchor(p Position) Position {
	center := l.at(p[1])
	depth := float32(cv(p).Sub(cv(center)).Dot(cv(l.Normal)))
	keep := float32(0)
	if over := p[1] - l.Top[1]; over > 0 {
		keep = min(1, over/.55)
	}
	pull := depth * (1 - keep)
	for a := range 3 {
		if a != 1 {
			p[a] -= l.Normal[a] * pull
		}
	}
	return p
}
