//go:build integration

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestPartyMemberBuffsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "party buffs", Width: 1024, Height: 768, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(true); err != nil {
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
	a.state.SelfID = 1
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Name: "Self", Level: 60}
	a.state.Pools.HP, a.state.Pools.MP, a.state.Pools.SP = 80, 60, 40
	a.state.Details.MaxHP, a.state.Details.MaxMP, a.state.Details.MaxSP = 100, 100, 100
	a.state.Party.Leader = 1
	effects := []proto.PartyEffect{{Skill: 86, Level: 1}, {Skill: 89, Level: 1}, {Skill: 181, Level: 1}, {Skill: 228, Level: 1}, {Skill: 84, Level: 1}, {Skill: 125, Level: 1}, {Skill: 87, Level: 1}}
	a.state.Party.Members = []proto.PartyMember{
		{ID: 1, Name: "Self", Level: 60, HP: 80, MaxHP: 100, MP: 60, MaxMP: 100, SP: 40, MaxSP: 100},
		{ID: 2, Name: "Friend", Level: 60, Class: 2, HP: 80, MaxHP: 100, MP: 60, MaxMP: 100, SP: 40, MaxSP: 100, Effects: effects},
		{ID: 3, Name: "Other", Level: 59, Class: 4, HP: 50, MaxHP: 100, MP: 60, MaxMP: 100, SP: 40, MaxSP: 100, Effects: effects[:2]},
	}
	gl.Viewport(0, 0, 1024, 768)
	gl.ClearColor(.2, .25, .3, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.clock = .5
	a.sprites.Begin(1024, 768)
	a.drawParty(1024, 768)
	a.sprites.End()
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if err := render.SavePNG(filepath.Join(dir, "party-buffs.png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUnionMenusGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "union", Width: 1024, Height: 768, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(true); err != nil {
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
	a.screen = screenWorld
	a.state.SelfID = 1
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Name: "Self", Level: 60}
	a.state.Pools.HP, a.state.Pools.MP, a.state.Pools.SP = 80, 60, 40
	a.state.Details.MaxHP, a.state.Details.MaxMP, a.state.Details.MaxSP = 100, 100, 100
	a.state.Party.Leader = 1
	for i := uint32(1); i <= 4; i++ {
		a.state.Party.Members = append(a.state.Party.Members, proto.PartyMember{ID: i, Name: fmt.Sprintf("Player %d", i), Level: 60, Class: uint8(i), HP: 80, MaxHP: 100, MP: 60, MaxMP: 100, SP: 40, MaxSP: 100})
	}
	shot := func(name string, setup func()) {
		setup()
		gl.Viewport(0, 0, 1024, 768)
		gl.ClearColor(.2, .25, .3, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(1024, 768)
		a.drawParty(1024, 768)
		a.drawPlayerContext(1024, 768)
		a.sprites.End()
		if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
			if err := render.SavePNG(filepath.Join(dir, name), 1024, 768); err != nil {
				t.Fatal(err)
			}
		}
	}
	shot("union-create.png", func() {
		a.performContextAction(1, contextAction{label: "Create union"})
	})
	shot("union-raid.png", func() {
		a.closeUnionConfig()
		a.state.Party.Raid, a.state.Party.RaidSubLeader = true, 2
		a.state.Party.RaidSlots = map[uint16]uint32{}
		a.state.Party.Members = nil
		for slot := uint16(0); slot < 9; slot++ {
			id := uint32(slot) + 1
			a.state.Party.RaidSlots[slot] = id
			a.state.Party.Members = append(a.state.Party.Members, proto.PartyMember{ID: id, Name: fmt.Sprintf("Player %02d", id), Level: 60, Class: uint8(slot % 6), HP: 80, MaxHP: 100, MP: 60, MaxMP: 100, SP: 40, MaxSP: 100})
		}
		a.syncUnionState(false)
		a.lastMouseX, a.lastMouseY = 130, 120
		a.state.Party.RaidAutoJoin = true
	})
}
