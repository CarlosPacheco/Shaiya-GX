package scene

import (
	"math"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// BenchmarkCloakFrame simulates one 60 Hz frame of a real human cloak while
// its wearer runs in a circle, without and with the native environment.
// GX_TEST_CLIENT_DATA names a loose client data directory.
func BenchmarkCloakFrame(b *testing.B) {
	dir := os.Getenv("GX_TEST_CLIENT_DATA")
	if dir == "" {
		b.Skip("requires loose client data")
	}
	base := filepath.Join(dir, "cloak", "Character", "Human")
	raw, err := os.ReadFile(filepath.Join(base, "1_humf.PC"))
	if err != nil {
		b.Fatal(err)
	}
	table, err := model.ParseCloakPC(raw)
	if err != nil {
		b.Fatal(err)
	}
	record := table.Records[0]
	raw, err = os.ReadFile(filepath.Join(base, "3DC", path.Base(table.Bodies[record.Body])))
	if err != nil {
		b.Fatal(err)
	}
	mesh, err := model.ParseCloth3DC(raw)
	if err != nil {
		b.Fatal(err)
	}
	rest := make([]render.V3, len(mesh.Vertices))
	for i, v := range mesh.Vertices {
		rest[i] = render.V3(v.Pos)
	}
	// Bones 1-5 stand up the spine; pins follow their own bone.
	finals := make([]render.M4, 8)
	for i := range finals {
		finals[i] = render.Identity()
		finals[i][13] = .9 + .12*float32(i)
	}
	SetClothGround(func(x, z float32) float32 { return 0 })
	defer SetClothGround(nil)
	for _, variant := range []struct {
		name    string
		env     bool
		collide bool
	}{{"previous", false, false}, {"wind", true, false}, {"native", true, true}} {
		b.Run(variant.name, func(b *testing.B) {
			p := newClothPhysics(rest, record.Columns, record.Rows, record.Mode)
			for _, pin := range record.Pins {
				p.pins[pin.Vertex] = true
			}
			c := &cloakInstance{physics: p, bodyType: 0}
			frame := 0
			for b.Loop() {
				angle := float64(frame) * .05
				world := render.Identity()
				world[0], world[2], world[8], world[10] = float32(math.Cos(angle)), float32(-math.Sin(angle)), float32(math.Sin(angle)), float32(math.Cos(angle))
				world[12], world[14] = float32(math.Sin(angle))*6, float32(math.Cos(angle))*6
				for _, pin := range record.Pins {
					p.positions[pin.Vertex] = transformPoint(render.V3(pin.Offset), finals[pin.Bone%len(finals)].Mul(world))
				}
				var env clothEnvironment
				if variant.env {
					env = c.environment(world, finals, variant.collide)
				}
				origin := render.V3{world[12], world[13], world[14]}
				for range 4 { // ceil(1/60 * 200)
					if !p.stepWithWind(.0005, env.wind) {
						p.reset(origin)
					}
					p.collide(&env)
				}
				frame++
			}
			b.ReportMetric(float64(len(rest)), "vertices")
		})
	}
}

func TestClothCollisionSpheresFloorAndBodyType(t *testing.T) {
	p := newClothPhysics([]render.V3{{0, 1, 0}, {0, 0, 0}}, 1, 2, 0)
	p.pins[0] = true
	e := clothEnvironment{count: 1, floor: -1, hasFloor: true}
	e.spheres[0], e.radii[0] = render.V3{0, 0, 0}, .25
	p.positions[1], p.velocity[1] = render.V3{.1, 0, 0}, render.V3{-1, 0, 0}
	p.collide(&e)
	if d := p.positions[1].Len(); math.Abs(float64(d)-.25) > 1e-5 || p.velocity[1][0] < 0 {
		t.Fatalf("sphere contact left %v moving %v", p.positions[1], p.velocity[1])
	}
	p.positions[1], p.velocity[1] = render.V3{2, -3, 0}, render.V3{0, -1, 0}
	p.collide(&e)
	if p.positions[1][1] != -1+clothFloorLift || p.velocity[1][1] != 0 {
		t.Fatalf("floor left %v", p.positions[1])
	}
	if got := clothBodyType(Appearance{Race: 1, Gender: 1, Job: 4}); got != 7 {
		t.Fatalf("elf woman mage body type %d", got)
	}
}
