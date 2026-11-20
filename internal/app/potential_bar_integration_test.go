//go:build integration

package app

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestPotentialCooldownDrawsOnAssignedBarsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	const width, height = 1024, 768
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: width, Height: height, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state = game.New()
	a.state.Skills[7] = proto.LearnedSkill{ID: 24, Level: 3, Number: 7}
	a.state.QuickSlots = []proto.QuickSlot{{Bar: 0, Slot: 3, Bag: 100, Number: 7}, {Bar: 1, Slot: 3, Bag: 100, Number: 7}}
	a.actionBar.HideSecond = false
	a.revolver.Position.Move(16, 280, width, height, 160, 111)
	a.revolver.Slots[0][0].Skill = 24

	x0, y0 := a.actionBar.SlotOrigin(3, width, height)
	x1, y1 := a.actionBar.SlotOrigin(13, width, height)
	r := a.revolverLayout(width, height).Cell(1, false)
	cells := [3][2]int{{x0, y0}, {x1, y1}, {r.X, r.Y}}
	draw := func() [3][]byte {
		gl.Viewport(0, 0, width, height)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(width, height)
		a.drawActionBar(width, height)
		a.drawRevolver(width, height)
		a.sprites.End()
		var out [3][]byte
		for i, cell := range cells {
			out[i] = make([]byte, 32*32*4)
			gl.ReadPixels(int32(cell[0]), int32(height-cell[1]-32), 32, 32, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&out[i][0]))
		}
		return out
	}
	ready := draw()
	a.clock = 100
	p := proto.NewWriter(proto.GXSkillCooldown)
	p.U16(24)
	p.U16(180)
	p.U16(180)
	a.applyWorldPacket(p.Bytes())
	active := draw()
	a.clock = 280
	expired := draw()
	for i := range cells {
		var red, yellow int
		for j := 0; j < len(active[i]); j += 4 {
			before, after := ready[i][j:j+3], active[i][j:j+3]
			if int(after[0])-int(before[0]) >= 12 && int(before[1])-int(after[1]) >= 8 {
				red++
			}
			if after[0] > 180 && after[1] > 170 && after[2] < 80 && !bytes.Equal(before, after) {
				yellow++
			}
		}
		if red < 100 || yellow < 10 || !bytes.Equal(ready[i], expired[i]) {
			t.Fatalf("assigned bar %d: red pixels %d, yellow timer pixels %d, cleared %t", i, red, yellow, bytes.Equal(ready[i], expired[i]))
		}
	}
}
