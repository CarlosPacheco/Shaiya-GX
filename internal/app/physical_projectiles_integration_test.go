//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"os"
	"runtime"
	"testing"
)

func TestOfficialRangedProjectilesGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("original resources and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 512, Height: 512, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	assets := scene.NewAssets(arc)
	renderer, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	renderer.View = render.LookAtLH(render.V3{4, 3, -8}, render.V3{4, 1, 0}, render.V3{0, 1, 0})
	renderer.Proj = render.PerspectiveFovLH(.785398, 1, .1, 100)
	for _, weapon := range []uint8{13, 14, 11} {
		a := inputApp()
		a.assets, a.scene3d = assets, renderer
		a.state.SelfID = 1
		actor, err := scene.BuildCharacter(assets, scene.Appearance{Race: 1, Job: 2, Height: 2, Helmet: -1, WeaponType: weapon}, "000_normal")
		if err != nil {
			t.Fatal(weapon, err)
		}
		actor.World = render.Identity()
		a.wv.self = actor
		a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindMob, X: 8}
		if !actor.PlayAction(scene.AttackAction(weapon, 0), false) {
			t.Fatal("ranged clip missing", weapon)
		}
		t.Log("ALT contact", weapon, actor.ContactTime())
		if !a.startPhysicalProjectile(proto.AttackResult{Source: 1, Target: 2, HP: 10}, weapon, actor) {
			t.Fatal("official projectile did not queue", weapon)
		}
		for i := 0; i < 240 && !a.wv.physicalProjectiles[0].released; i++ {
			actor.Update(1.0 / 120)
			a.updatePhysicalProjectiles(1.0 / 120)
			if len(a.wv.physicalProjectiles) == 0 {
				t.Fatal("shot cancelled before release", weapon)
			}
		}
		if !a.wv.physicalProjectiles[0].released {
			t.Fatal("release phase never reached", weapon)
		}
		gl.Viewport(0, 0, 512, 512)
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.drawPhysicalProjectiles()
		pixels := make([]byte, 512*512*4)
		gl.ReadPixels(0, 0, 512, 512, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		visible := 0
		for i := 0; i < len(pixels); i += 4 {
			if pixels[i] > 5 || pixels[i+1] > 5 || pixels[i+2] > 5 {
				visible++
			}
		}
		if visible < 3 {
			t.Fatal("projectile model rendered no pixels", weapon, visible)
		}
		speed := a.wv.physicalProjectiles[0].speed
		actor.ReleaseAction()
		a.state.Entities[2].X = 11
		a.updatePhysicalProjectiles(1.0 / 120)
		if len(a.wv.physicalProjectiles) == 0 || a.wv.physicalProjectiles[0].speed != speed {
			t.Fatal("new action/target movement cancelled launched shot", weapon)
		}
		for i := 0; i < 120 && len(a.wv.physicalProjectiles) > 0; i++ {
			a.updatePhysicalProjectiles(1.0 / 120)
		}
		if len(a.wv.physicalProjectiles) != 0 || len(a.combatNumbers) != 1 {
			t.Fatal("shot failed to arrive", weapon, len(a.combatNumbers))
		}
		t.Log("official ranged model rendered and arrived", weapon, "pixels", visible)
		// Physical ranged skills use the same arrow/weapon flight rather than
		// showing their target impact and number before the missile arrives.
		a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 57, Level: 1}: {Name: "Ranged probe", Attack: 2, Effect: 1, Target: 3}}
		a.state.Inventory[[2]uint8{0, 5}] = proto.InventoryItem{Type: weapon}
		actor.PlayAction(scene.AttackAction(weapon, 0), false)
		a.state.LastSkill = proto.SkillResult{Source: 1, Target: 2, ID: 57, Level: 1, HP: 10}
		a.confirmedSkillFeedback(proto.SkillUseNoTarget)
		if !a.wv.physicalSkillQueued || len(a.wv.physicalProjectiles) != 1 || len(a.combatNumbers) != 1 {
			t.Fatal("physical skill did not defer recipient feedback", weapon)
		}
		a.startSkillResultEffects(a.state.LastSkill)
		if len(a.wv.impacts) != 0 {
			t.Fatal("target effect appeared before physical skill flight", weapon)
		}
		// The fixture has no complete App/world render loop; advance its real actor.
		for i := 0; i < 360 && len(a.wv.physicalProjectiles) > 0; i++ {
			actor.Update(1.0 / 120)
			a.updatePhysicalProjectiles(1.0 / 120)
		}
		if len(a.wv.physicalProjectiles) != 0 || len(a.combatNumbers) != 2 {
			t.Fatal("physical skill impact/number did not arrive", weapon, len(a.combatNumbers))
		}

	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("projectile GL error %#x", code)
	}
}
