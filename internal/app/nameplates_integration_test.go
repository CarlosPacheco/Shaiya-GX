//go:build integration

package app

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

func TestNPCNameplateOriginalAssetsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "NPC nameplate", Width: 1024, Height: 734, Hidden: true, Debug: true})
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
	if err := a.Preview("world-stream"); err != nil {
		t.Fatal(err)
	}
	a.state.Guild.ID, a.state.Guild.Name = 7, "Guardians"
	right := render.V3{0, 1, 0}.Cross(a.wv.facing)
	pos := a.wv.pos.Add(a.wv.facing.Scale(4)).Add(right.Scale(3))
	pos[1] = a.wv.world.Height(pos[0], pos[2])
	if pos[1] > 1 {
		drop := proto.GroundItem{X: pos[0], Y: 0, Z: pos[2]}
		if got := groundItemLabelPosition(a.wv.world, drop); got[1] != pos[1]+.3 || drop.Y != 0 {
			t.Fatalf("ground drop presentation should follow terrain without changing server data: %v", got)
		}
		drop.Y = pos[1] + 5
		if got := groundItemLabelPosition(a.wv.world, drop); got[1] != drop.Y+.3 {
			t.Fatalf("elevated server drop position was lost: %v", got)
		}
	} else {
		t.Fatal("NPC fixture did not provide elevated terrain for the item label")
	}
	a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: 1, TypeID: 6, X: pos[0], Y: pos[1], Z: pos[2]}
	for i := 0; i < 240; i++ {
		gl.Viewport(0, 0, 1024, 734)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.Draw(1024, 734)
		if a.SceneReady() {
			break
		}
	}
	if !a.SceneReady() {
		t.Fatal("field did not finish streaming")
	}
	actor := a.wv.npcs[42]
	if actor == nil || actor.actor == nil || actor.actor.Name == "" {
		t.Fatal("original NPC was not rendered")
	}

	// Ordinary NPCs retain server height; GM summons follow the same ground
	// surface as the player even if their packet uses an elevated caster Y.
	entity := a.state.Entities[42]
	originalPosition := actor.position
	entity.Y = pos[1] + 100
	actor.position = remotePosition{}
	a.prepareNPCs()
	if actor.position.pos[1] != entity.Y {
		t.Fatal("ordinary NPC lost server height")
	}
	entity.Summoned = true
	a.prepareNPCs()
	if actor.position.pos[1] != pos[1] || entity.Y != pos[1]+100 {
		t.Fatal("summoned NPC did not follow terrain")
	}
	entity.Summoned = false
	entity.Y = pos[1]
	actor.position = originalPosition
	a.prepareNPCs()
	anchor := actor.actor.NPCPlateAnchor(entity.NpcType, entity.TypeID)
	if anchor[0] != pos[0] || anchor[2] != pos[2] || anchor[1] <= pos[1] {
		t.Fatalf("NPC label did not use its world origin and authored height: %v", anchor)
	}
	if a.assets.Texture("interface/mon_not.png") == nil {
		t.Fatal("original neutral label marker unavailable")
	}
	center := a.visualEntityPosition(a.state.Entities[42]).Add(render.V3{0, 1, 0})
	mx, my, ok := projectName(center, a.scene3d.View.Mul(a.scene3d.Proj), 1024, 734)
	if !ok || a.pickNPC(&platform.Input{Clicked: true, MouseX: mx, MouseY: my}, 1024, 734) != 42 {
		t.Fatal("displayed NPC cannot be selected by mouse")
	}
	blocker, err := scene.BuildCharacter(a.assets, scene.Appearance{Helmet: -1}, "000_normal")
	if err != nil {
		t.Fatal(err)
	}
	blockingPos := a.wv.eye.Add(center.Sub(a.wv.eye).Scale(.65)).Sub(render.V3{0, 1, 0})
	blocker.World = render.Basis(blockingPos, render.V3{0, 0, -1}, render.V3{0, 1, 0})
	a.state.Entities[100] = &game.Entity{ID: 100, Kind: game.KindUser, Shape: &proto.UserShape{ID: 100}}
	a.wv.actors[100] = &remoteActor{entity: a.state.Entities[100], character: blocker}
	if a.pickNPC(&platform.Input{Clicked: true, MouseX: mx, MouseY: my}, 1024, 734) != 0 {
		t.Fatal("NPC behind another actor was selected")
	}
	if a.pickMob(&platform.Input{Clicked: true, MouseX: mx, MouseY: my}, 1024, 734) != 100 {
		t.Fatal("front player was not selected over NPC")
	}
	delete(a.state.Entities, 100)
	delete(a.wv.actors, 100)
	monster, err := a.assets.BuildMonster(2009)
	if err != nil {
		t.Fatal(err)
	}
	monster.World = blocker.World
	lo, hi, valid := monster.WorldBounds()
	if !valid {
		t.Fatal("monster bounds unavailable")
	}
	delta := a.wv.eye.Add(center.Sub(a.wv.eye).Scale(.65)).Sub(lo.Add(hi).Scale(.5))
	monster.World[12] += delta[0]
	monster.World[13] += delta[1]
	monster.World[14] += delta[2]
	a.state.Entities[101] = &game.Entity{ID: 101, Kind: game.KindMob}
	if a.wv.mobs == nil {
		a.wv.mobs = map[uint32]*mobActor{} // created by the first monster update in play
	}
	a.wv.mobs[101] = &mobActor{entity: a.state.Entities[101], actor: monster, pos: render.V3{monster.World[12], monster.World[13], monster.World[14]}}
	if a.pickMob(&platform.Input{Clicked: true, MouseX: mx, MouseY: my}, 1024, 734) != 101 {
		t.Fatal("front monster was not selected over NPC")
	}
	entities, target, occlusion := a.state.Entities, a.state.Target, a.scene3d.Occlusion
	a.state.Target, a.scene3d.Occlusion = 0, nil
	renderPlates := func(showIDs bool) []byte {
		a.showCatalogIDs = showIDs
		gl.DepthFunc(gl.LEQUAL)
		gl.DepthMask(true)
		gl.ClearColor(0, 0, 0, 1)
		gl.ClearDepth(1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(1024, 734)
		a.drawPlayerNames(1024, 734)
		a.sprites.End()
		pixels := make([]byte, 1024*734*4)
		gl.ReadPixels(0, 0, 1024, 734, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		return pixels
	}
	entities[101].TypeID = 2009
	a.state.Entities = map[uint32]*game.Entity{101: entities[101]}
	if !bytes.Equal(renderPlates(false), renderPlates(true)) {
		t.Fatal("debug IDs revealed an unselected monster's nameplate")
	}
	a.state.Target = 101
	if bytes.Equal(renderPlates(false), renderPlates(true)) {
		t.Fatal("the selected monster's nameplate did not get its catalog ID")
	}
	a.state.Target = 0
	a.state.Entities = map[uint32]*game.Entity{42: entities[42]}
	if bytes.Equal(renderPlates(false), renderPlates(true)) {
		t.Fatal("an unselected NPC did not show its catalog ID")
	}
	a.state.Entities, a.state.Target, a.scene3d.Occlusion = entities, target, occlusion
	a.showCatalogIDs = false
	gl.DepthMask(true)
	gl.ClearDepth(0)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	a.sprites.CaptureNameDepth(1024, 734)
	gl.ClearDepth(1)
	if a.pickNPC(&platform.Input{Clicked: true, MouseX: mx, MouseY: my}, 1024, 734) != 0 {
		t.Fatal("NPC behind closer depth was selected")
	}
	if a.pickMob(&platform.Input{Clicked: true, MouseX: mx, MouseY: my}, 1024, 734) != 0 {
		t.Fatal("player behind closer depth was selected")
	}
	delete(a.state.Entities, 101)
	delete(a.wv.mobs, 101)
	move := proto.NewWriter(proto.MapNpcMove)
	move.U32(42)
	move.U8(0)
	move.F32(pos[0] + 4)
	move.F32(pos[1])
	move.F32(pos[2])
	if _, err := a.state.Apply(move.Bytes()); err != nil {
		t.Fatal(err)
	}
	a.clock += 1
	a.prepareNPCs()
	if actor.actor.World[12] != pos[0] {
		t.Fatal("NPC snapped to patrol destination")
	}
	a.clock = actor.position.start + actor.position.duration*.5
	a.prepareNPCs()
	if actor.actor.World[12] != pos[0]+2 || a.state.Entities[42].X != pos[0]+4 {
		t.Fatal("NPC patrol interpolation did not preserve server position")
	}
	if got := a.visualEntityPosition(a.state.Entities[42]); got[0] != pos[0]+2 {
		t.Fatal("NPC effect did not follow model")
	}
	a.clock = actor.position.start + actor.position.duration
	a.prepareNPCs()
	if actor.actor.World[12] != pos[0]+4 {
		t.Fatal("NPC did not reach patrol destination")
	}
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		a.Draw(1024, 734)
		if err := render.SavePNG(filepath.Join(dir, "npc-nameplate-preview.png"), 1024, 734); err != nil {
			t.Fatal(err)
		}
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("NPC nameplate GL error %#x", code)
	}
}

func TestNameDepthCaptureIsLazyGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "name depth laziness", Width: 64, Height: 64, Hidden: true})
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
	a.wv = &worldView{}
	a.items = map[sdata.ItemKey]*sdata.Item{}
	a.scene3d.View, a.scene3d.Proj = render.Identity(), render.Identity()
	// A known entity without a renderable shape should not allocate depth.
	a.state.Entities[1] = &game.Entity{ID: 1, Kind: game.KindUser}
	gl.Viewport(0, 0, 64, 64)
	a.sprites.Begin(64, 64)
	before := render.GPUBytes()
	a.drawPlayerNames(64, 64)
	a.sprites.End()
	if got := render.GPUBytes(); got != before {
		t.Fatalf("empty name pass allocated GPU storage: %d -> %d", before, got)
	}
	// Ground items have no nameplates. Without a visible model there must be
	// no label drawing and no lazy depth allocation for an invisible text hitbox.
	a.state.GroundItems[2] = proto.GroundItem{Type: 26, X: 0, Y: 0, Z: .5}
	a.sprites.Begin(64, 64)
	a.drawPlayerNames(64, 64)
	a.sprites.End()
	if got := render.GPUBytes(); got != before {
		t.Fatal("ground item text allocated name depth", before, got)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("lazy name depth GL error %#x", code)
	}
}

func TestOriginalBossPickWidthLeavesOtherMonsterSelectableGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original monster assets and OpenGL")
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
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	assets := scene.NewAssets(arc)
	boss, err := assets.BuildMonster(872) // Cloron Dragon
	if err != nil {
		t.Fatal(err)
	}
	monkey, err := assets.BuildMonster(2009)
	if err != nil {
		t.Fatal(err)
	}

	boss.SetTransform(render.Basis(render.V3{0, 0, 10}, render.V3{0, 0, -1}, render.V3{0, 1, 0}))
	height := boss.MonsterPlateAnchor()[1] / 2
	camera := &render.Scene3D{
		View: render.LookAtLH(render.V3{0, height, -10}, render.V3{0, height, 10}, render.V3{0, 1, 0}),
		Proj: render.PerspectiveFovLH(math.Pi/4, 1, .1, 100),
	}
	a := &App{state: game.New(), scene3d: camera, wv: &worldView{mobs: make(map[uint32]*mobActor)}}
	bossEntity := &game.Entity{ID: 101, Kind: game.KindMob, Z: 10}
	a.state.Entities[101] = bossEntity
	a.wv.mobs[101] = &mobActor{entity: bossEntity, actor: boss}

	const size = 512
	center := &platform.Input{Clicked: true, MouseX: size / 2, MouseY: size / 2}
	if got := a.pickMob(center, size, size); got != bossEntity.ID {
		t.Fatalf("click within the original rectangle selected %d", got)
	}
	lo, hi, ok := boss.WorldBounds()
	if !ok {
		t.Fatal("missing dragon mesh bounds")
	}
	vp := camera.View.Mul(camera.Proj)
	for x := size/2 + 1; x < size; x++ {
		origin, direction, ok := clickRay(x, size/2, size, size, vp)
		if !ok {
			continue
		}
		front, boxHit := rayBox(origin, direction, lo, hi)
		_, quadHit := a.monsterRayHit(boss, origin, direction)
		if !boxHit || quadHit {
			continue
		}
		pos := origin.Add(direction.Scale(front + 10))
		pos[1] -= .25
		monkey.SetTransform(render.Basis(pos, render.V3{0, 0, -1}, render.V3{0, 1, 0}))
		other := &game.Entity{ID: 102, Kind: game.KindMob, X: pos[0], Y: pos[1], Z: pos[2]}
		a.state.Entities[other.ID] = other
		a.wv.mobs[other.ID] = &mobActor{entity: other, actor: monkey}
		if got := a.pickMob(&platform.Input{Clicked: true, MouseX: x, MouseY: size / 2}, size, size); got != other.ID {
			t.Fatalf("dragon envelope stole click from visible monster: got %d", got)
		}
		return
	}
	t.Fatal("dragon fixture did not provide a ray outside its original selection rectangle")
}
