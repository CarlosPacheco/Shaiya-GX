package scene

import (
	"slices"

	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type clothSample struct {
	upper, lower int
	fraction     float32
	// Preserve the original folds at rest as the grid interpolates motion.
	restOffset render.V3
}

func (s clothSample) blend(values []render.V3) render.V3 {
	a, b := values[s.upper], values[s.lower]
	return a.Add(b.Sub(a).Scale(s.fraction))
}

// A cloak geometry shares one reduced grid. Actor positions and velocities
// still evolve separately on that grid.
type clothGrid struct {
	nodes    []int
	samples  []clothSample
	pins     []model.CloakPin
	template *clothPhysics
}

func makeClothGrid(record model.CloakRecord, vertices []render.Vertex) *clothGrid {
	// Keep the authored cross section and all anchors across the shoulders.
	columns := record.Columns
	keep := make([]bool, record.Rows)
	for row := range keep {
		keep[row] = row%2 == 0 || row == record.Rows-1
	}
	for _, pin := range record.Pins {
		keep[pin.Vertex/columns] = true
	}
	var rows []int
	for row, retained := range keep {
		if retained {
			rows = append(rows, row)
		}
	}
	if len(rows)*4 > record.Rows*3 {
		return nil
	}
	g := &clothGrid{samples: make([]clothSample, len(vertices))}
	rest := make([]render.V3, 0, columns*len(rows))
	for _, row := range rows {
		for col := range columns {
			i := row*columns + col
			g.nodes = append(g.nodes, i)
			rest = append(rest, vertices[i].Pos)
		}
	}
	g.template = newClothPhysics(rest, columns, len(rows), record.Mode)
	for _, pin := range record.Pins {
		pin.Vertex, _ = slices.BinarySearch(g.nodes, pin.Vertex)
		g.pins = append(g.pins, pin)
		g.template.pins[pin.Vertex] = true
	}
	for row := range record.Rows {
		upper, lower, fraction := clothGridSpan(rows, row)
		for col := range columns {
			i := row*columns + col
			s := clothSample{upper: upper*columns + col, lower: lower*columns + col, fraction: fraction}
			s.restOffset = vertices[i].Pos.Sub(s.blend(rest))
			g.samples[i] = s
		}
	}
	return g
}

func clothGridSpan(axis []int, point int) (int, int, float32) {
	to, exact := slices.BinarySearch(axis, point)
	if exact {
		return to, to, 0
	}
	from := to - 1
	return from, to, float32(point-axis[from]) / float32(axis[to]-axis[from])
}

func (g *clothGrid) simulation() *clothPhysics {
	p := *g.template
	p.positions = make([]render.V3, len(p.rest))
	p.velocity = make([]render.V3, len(p.rest))
	p.forces = make([]render.V3, len(p.rest))
	p.springForces = make([]render.V3, len(p.springForces))
	return &p
}
