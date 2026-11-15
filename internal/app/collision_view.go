package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"log"
	"math"
	"strings"
)

func (a *App) collisionViewCommand(text string) bool {
	fields := strings.Fields(text)
	if len(fields) == 0 || !strings.EqualFold(fields[0], "/cview") {
		return false
	}
	if a.state == nil {
		return true
	}
	if !a.state.Self.IsGM {
		a.questMessage("GM: access denied.")
		return true
	}
	if a.wv == nil {
		return true
	}
	if len(fields) > 2 || len(fields) == 2 && fields[1] != "on" && fields[1] != "off" {
		a.questMessage("Usage: /cview [on|off]")
		return true
	}
	enabled := !a.wv.collisionView
	if len(fields) == 2 {
		enabled = fields[1] == "on"
	}
	a.wv.collisionView = enabled
	if enabled {
		a.questMessage("Collision view ON: gold = walls, cyan = support surfaces, blue = terrain, magenta = player ellipsoid. Client prediction only.")
	} else {
		a.questMessage("Collision view OFF.")
	}
	return true
}
func (a *App) drawCollisionView() {
	v := a.wv
	if v == nil || !v.collisionView {
		return
	}
	if a.state == nil || !a.state.Self.IsGM {
		v.collisionView = false
		return
	}
	if a.collisionLines == nil {
		var err error
		a.collisionLines, err = render.NewDebugLines()
		if err != nil {
			log.Printf("collision view: %v", err)
			v.collisionView = false
			return
		}
	}
	vertices := a.collisionVertices[:0]
	line := func(start, end render.V3, color [4]float32) {
		vertices = append(vertices, render.DebugLineVertex{Position: start, Color: color}, render.DebugLineVertex{Position: end, Color: color})
	}
	vector := func(p collision.Vec) render.V3 { return render.V3{float32(p[0]), float32(p[1]), float32(p[2])} }
	count := 0
	triangle := func(t collision.Triangle, color [4]float32) {
		if count >= 15000 {
			return
		}
		count++
		for i := range t {
			line(vector(t[i]), vector(t[(i+1)%3]), color)
		}
	}
	v.world.CollisionTriangles(v.pos, 40, func(t collision.Triangle) {
		normal := t[1].Sub(t[0]).Cross(t[2].Sub(t[0])).Norm()
		color := [4]float32{1, .65, .1, .75}
		if normal[1] > .5 {
			color = [4]float32{.1, 1, 1, .75}
		}
		triangle(t, color)
	})
	v.world.TerrainCollisionTriangles(v.pos, 24, func(t collision.Triangle) { triangle(t, [4]float32{.2, .45, 1, .5}) })
	centre := v.pos
	centre[1] += float32(collision.PlayerRadii[1])
	color := [4]float32{1, .2, 1, 1}
	for axis := range 3 {
		for i := 0; i < 48; i++ {
			point := func(j int) render.V3 {
				angle := float64(j) * 2 * math.Pi / 48
				p := centre
				a, b := (axis+1)%3, (axis+2)%3
				p[a] += float32(math.Cos(angle) * collision.PlayerRadii[a])
				p[b] += float32(math.Sin(angle) * collision.PlayerRadii[b])
				return p
			}
			line(point(i), point(i+1), color)
		}
	}
	a.collisionVertices = vertices
	a.collisionLines.Draw(a.scene3d, vertices)
}
