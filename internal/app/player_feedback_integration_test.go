//go:build integration

package app

import (
	"math"
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestNativePlayerFeedbackGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original effect libraries and OpenGL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 320, Height: 240, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	assets := scene.NewAssets(arc)
	defer assets.CloseCharacterResources()
	a := &App{state: game.New(), assets: assets, wv: &worldView{mapID: 2}}
	a.state.MapID, a.state.SelfID, a.state.Phase = 2, 1, game.PhaseInWorld
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Level: 10}
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindUser, X: 5, Y: 2, Z: 3,
		Shape: &proto.UserShape{Faction: 1}}
	a.applyWorldPacket(proto.PlayerFeedbackPacket(2, proto.PlayerSpawnFeedback))
	a.state.Entities[2].X = 7
	a.applyWorldPacket(proto.PlayerFeedbackPacket(2, proto.PlayerSkillFeedback))

	level := func(op proto.Opcode, id uint32, rank uint16) {
		p := proto.NewWriter(op)
		p.U32(id)
		p.U16(rank)
		p.U16(0)
		p.U16(0)
		p.U32(0)
		p.U32(0)
		a.applyWorldPacket(p.Bytes())
	}
	level(proto.LevelUpSelf, 1, 10) // GM stat point notification.
	level(proto.LevelUpSelf, 1, 11)
	level(proto.LevelUpOther, 2, 11)
	a.applyWorldPacket(proto.PlayerFeedbackPacket(1, proto.PlayerLevelFeedback))
	a.applyWorldPacket(proto.PlayerFeedbackPacket(2, proto.PlayerLevelFeedback))

	a.frameDT = .05
	a.updateResolvedEffects()
	if len(a.wv.impacts) != 6 || len(a.wv.effectErrors) != 0 {
		t.Fatalf("effects = %d, errors = %v", len(a.wv.impacts), a.wv.effectErrors)
	}
	m := a.wv.impacts[0].fixedWorld
	if m == nil || [3]float32{m[12], m[13], m[14]} != [3]float32{5, 2, 3} {
		t.Fatal("spawn anchor moved with the player")
	}
	scene3d, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer scene3d.Delete()
	scene3d.Proj = render.PerspectiveFovLH(math.Pi/3, 320.0/240, .1, 100)
	gl.Viewport(0, 0, 320, 240)
	gl.ClearColor(0, 0, 0, 1)
	for range 22 {
		a.updateResolvedEffects()
	}
	for _, impact := range a.wv.impacts {
		x, y, z := impact.instance.Origin[0], impact.instance.Origin[1], impact.instance.Origin[2]
		scene3d.View = render.LookAtLH(render.V3{x, y + 2, z - 7}, render.V3{x, y + 1, z}, render.V3{0, 1, 0})
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		impact.instance.Draw(scene3d)
		pixels := make([]byte, 320*240*4)
		gl.ReadPixels(0, 0, 320, 240, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		visible := 0
		for i := 0; i < len(pixels); i += 4 {
			if pixels[i]|pixels[i+1]|pixels[i+2] != 0 {
				visible++
			}
		}
		if visible == 0 {
			t.Fatal("original effect rendered no pixels")
		}
		t.Logf("effect %d at %v renders %d pixels", impact.target, impact.instance.Origin, visible)
	}
	for range 240 {
		a.updateResolvedEffects()
	}
	if len(a.wv.impacts) != 0 {
		t.Fatal("player feedback outlived its presentation")
	}
}
