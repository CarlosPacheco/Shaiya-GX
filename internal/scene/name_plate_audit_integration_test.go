//go:build integration

package scene

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// TestNamePlateAudit measures every monster and NPC model against the height
// its name plate is placed at. Set NAME_PLATE_AUDIT=file.tsv to keep the rows:
//
//	kind id name monHeight meshTop meshBottom plateY fittedPlateY bodyTop
func TestNamePlateAudit(t *testing.T) {
	out := os.Getenv("NAME_PLATE_AUDIT")
	if out == "" || os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets, GL and NAME_PLATE_AUDIT")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	assets := NewAssets(arc)
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	origin := render.Basis(render.V3{}, render.V3{0, 0, -1}, render.V3{0, 1, 0})
	row := func(kind string, id int, name string, m *CreatureActor, plate, fitted float32) {
		lo, hi, ok := m.WorldBounds()
		if ok {
			top, _ := m.bodyTop()
			fmt.Fprintf(f, "%s\t%d\t%s\t%.3f\t%.3f\t%.3f\t%.3f\t%.3f\t%.3f\n", kind, id, name, m.visual.Height*m.modelScale(), hi[1], lo[1], plate, fitted, top)
		}
	}
	for id := 0; ; id++ {
		def, err := assets.MonsterDefinition(uint16(id))
		if err != nil {
			break
		}
		m, err := assets.BuildMonster(uint16(id))
		if err != nil || m == nil {
			continue
		}
		m.SetTransform(origin)
		m.Update(0)
		base := m.MonsterPlateAnchor()
		row("mob", id, def.Name, m, base[1], m.FitPlate(base, .12)[1])
	}
	raw, _ := arc.ReadFile("Npc/NpcQuest.SData")
	defs, _ := sdata.ParseNPCs(raw)
	for key, d := range defs {
		m, err := assets.BuildNPC(d)
		if err != nil || m == nil {
			continue
		}
		m.SetTransform(origin)
		m.Update(0)
		base := m.NPCPlateAnchor(key.Type, key.ID)
		fitted := base
		if key.Type != 12 && !(key.Type == 7 && key.ID >= 541 && key.ID <= 543) {
			fitted = m.FitPlate(base, .15)
		}
		row(fmt.Sprintf("npc%d", key.Type), int(key.ID), d.Name, m, base[1], fitted[1])
	}
}
