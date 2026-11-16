//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/audio"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestVehicleAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_VEHICLE") != "1" || os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires disposable mount and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "mount probe", Width: 1024, Height: 734, Hidden: true, Debug: true})
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
	st, world, _, wait := testworld.Enter(t)
	other, otherWorld, _, waitOther := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	wait(func() bool { return st.Entities[other.SelfID] != nil })
	waitOther(func() bool { return other.Entities[st.SelfID] != nil })
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state, a.world, a.screen = st, world, screenWorld
	a.enterWorld()
	artifactDir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if artifactDir == "" {
		artifactDir = t.TempDir()
	}
	for deadline := time.Now().Add(25 * time.Second); time.Now().Before(deadline); {
		a.Update(&platform.Input{}, 1024, 734, 1.0/60)
		gl.Viewport(0, 0, 1024, 734)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.Draw(1024, 734)
		if a.SceneReady() {
			break
		}
		time.Sleep(time.Millisecond)
	}
	item := st.Inventory[[2]uint8{0, 13}]
	if item.Type != 42 || item.Count != 1 || st.Self.Mounted {
		t.Fatal("invalid mount fixture")
	}
	a.inventory.open = true
	ix, iy := a.inventoryUI.Origin(1024, 734)
	sx, sy, _, _ := ui.InventoryCell(0, 13)
	a.worldControls(&platform.Input{MouseX: ix + sx + 2, MouseY: iy + sy + 2, Pressed: true, MouseDown: true}, 1024, 734)
	a.worldControls(&platform.Input{MouseX: ix + sx + 2, MouseY: iy + sy + 2, Clicked: true}, 1024, 734)
	a.worldControls(&platform.Input{Chars: []rune{'u'}}, 1024, 734)
	if !a.vehicle.pending || st.Self.Mounted {
		t.Fatal("mount was applied before server acknowledgement")
	}
	sawCast := false
	deadline := time.Now().Add(10 * time.Second)
	for a.vehicle.pending || !st.Self.Mounted || st.Self.MountShape != 1 {
		a.pumpWorld()
		if a.state != nil && a.state.Self.Vehicle.State == proto.VehicleCasting && !sawCast {
			_, total, active := a.wv.self.CastTiming()
			if !active || total < 1.9 || total > 2 || len(a.wv.castEffects) != 2 {
				t.Fatal("mount did not use two-second cast bar and Town Portal effect", total, active)
			}
			sawCast = true
			a.inventory.open = false
			a.questLog.open = false
			a.Update(&platform.Input{}, 1024, 734, .5)
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			a.Draw(1024, 734)
			if err := render.SavePNG(filepath.Join(artifactDir, "vehicle-cast-hands.png"), 1024, 734); err != nil {
				t.Fatal(err)
			}
		}
		if a.state == nil || time.Now().After(deadline) {
			t.Fatal("mount confirmation timed out")
		}
		time.Sleep(time.Millisecond)
	}
	if len(a.wv.impacts) == 0 {
		t.Fatal("mount appearance transformation missing")
	}
	a.Update(&platform.Input{}, 1024, 734, .2)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.Draw(1024, 734)
	if err := render.SavePNG(filepath.Join(artifactDir, "vehicle-transformation.png"), 1024, 734); err != nil {
		t.Fatal(err)
	}
	if !sawCast {
		t.Fatal("server skipped mount preparation")
	}
	waitOther(func() bool {
		e := other.Entities[st.SelfID]
		return e != nil && e.Shape != nil && e.Mounted && e.MountShape == 1
	})
	a.inventory.open = false
	a.questLog.open = false
	for deadline := time.Now().Add(25 * time.Second); time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
		a.Update(&platform.Input{}, 1024, 734, 1.0/60)
		gl.Viewport(0, 0, 1024, 734)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.Draw(1024, 734)
		if a.SceneReady() && a.wv.selfMount != nil {
			break
		}
	}
	if !a.SceneReady() || a.wv.selfMount == nil || !a.wv.self.Mounted {
		t.Fatalf("mount model or rider pose missing: ready=%v loading=%v mount=%v characterMounted=%v state=%+v screen=%v", a.SceneReady(), a.loading.active, a.wv.selfMount != nil, a.wv.self.Mounted, a.state.Self.Vehicle, a.screen)
	}
	seat := a.wv.selfMount.Seat()
	if seat[1]-a.wv.pos[1] < .2 || seat[1]-a.wv.pos[1] > 4 {
		t.Fatalf("implausible seat height: %+v rider=%+v", seat, a.wv.pos)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("mount GL error %#x", code)
	}
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := render.SavePNG(filepath.Join(dir, "vehicle-preview.png"), 1024, 734); err != nil {
		t.Fatal(err)
	}
	beforeMove := a.wv.pos
	a.Update(&platform.Input{Down: "W"}, 1024, 734, .1)
	if a.wv.pos == beforeMove || !a.wv.moving {
		t.Fatal("mounted character could not move")
	}
	a.Draw(1024, 734)
	beforeBack := a.wv.pos
	a.Update(&platform.Input{Down: "S"}, 1024, 734, .1)
	a.Draw(1024, 734)
	if a.wv.pos == beforeBack || a.wv.visualFacing.Len() > 0 {
		t.Fatal("backing up rotated the mount")
	}
	beforeSide := a.wv.pos
	a.Update(&platform.Input{Down: "E"}, 1024, 734, .1)
	sideDelta := a.wv.pos.Sub(beforeSide)
	sideDelta[1] = 0
	if sideDelta.Len() < .7 {
		t.Fatal("mounted lateral movement was slowed", sideDelta.Len())
	}
	a.Update(&platform.Input{}, 1024, 734, .01)
	a.Update(&platform.Input{Keys: []platform.Key{platform.KeySpace}}, 1024, 734, 1.0/120)
	if !a.wv.jumping || !st.Self.Mounted {
		t.Fatal("driver could not jump while mounted")
	}
	for frame := 0; frame < 180 && a.wv.jumping; frame++ {
		time.Sleep(time.Second / 120)
		a.Update(&platform.Input{}, 1024, 734, 1.0/120)
		a.Draw(1024, 734)
	}
	if a.wv.jumping || !st.Self.Mounted {
		t.Fatal("mounted jump did not land")
	}
	a.Update(&platform.Input{}, 1024, 734, .01)
	observer, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	observer.state, observer.world, observer.screen = other, otherWorld, screenWorld
	observer.enterWorld()
	for deadline := time.Now().Add(25 * time.Second); time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
		gl.Viewport(0, 0, 1024, 734)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		observer.Draw(1024, 734)
		if observer.SceneReady() && observer.wv.actors[st.SelfID] != nil && observer.wv.actors[st.SelfID].mount != nil {
			break
		}
	}
	if observer.wv.actors[st.SelfID] == nil || observer.wv.actors[st.SelfID].mount == nil {
		t.Fatal("remote player's mount model missing")
	}
	// Native context action and confirmation UI attach the second rider.
	var together *contextAction
	actions := a.contextActions(other.SelfID)
	for i := range actions {
		if actions[i].label == "Go together" {
			together = &actions[i]
			break
		}
	}
	if together == nil {
		t.Fatal("go together action missing on two-seat mount")
	}
	a.performContextAction(other.SelfID, *together)
	waitOther(func() bool { return other.VehicleInviteID == st.SelfID })
	observer.updateVehicleInvite(&platform.Input{}, 1024, 734)
	if !observer.vehicleMessage.Visible {
		t.Fatal("native invitation prompt missing")
	}
	observer.updateVehicleInvite(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 734)
	wait(func() bool { return st.Self.Vehicle.Passenger == other.SelfID })
	waitOther(func() bool { return other.Self.Vehicle.State == proto.VehiclePassenger })
	observer.Draw(1024, 734)
	if err := render.SavePNG(filepath.Join(dir, "vehicle-passenger.png"), 1024, 734); err != nil {
		t.Fatal(err)
	}
	if observer.wv.selfMount != nil {
		t.Fatal("passenger drew a duplicate mount")
	}
	mount := observer.wv.actors[st.SelfID].mount
	expected := mount.RiderSeat(true)
	actual := render.V3{observer.wv.self.World[12], observer.wv.self.World[13], observer.wv.self.World[14]}
	if actual.Sub(expected).Len() > .001 {
		t.Fatal("passenger did not use secondary bone", actual, expected)
	}
	observer.Update(&platform.Input{Keys: []platform.Key{platform.KeySpace}}, 1024, 734, .01)
	waitOther(func() bool { return other.Self.Vehicle.State == proto.VehicleOff })
	wait(func() bool { return st.Self.Vehicle.Passenger == 0 })
	observer.world = nil
	observer.Close()
	if !a.vehicleCommand("/mount") || !a.vehicle.pending || !st.Self.Mounted {
		t.Fatal("dismount changed state optimistically")
	}
	deadline = time.Now().Add(10 * time.Second)
	for a.vehicle.pending || st.Self.Mounted || st.Self.MountShape != 0 {
		a.pumpWorld()
		if a.state == nil || time.Now().After(deadline) {
			t.Fatal("dismount confirmation timed out")
		}
		time.Sleep(time.Millisecond)
	}
	waitOther(func() bool { e := other.Entities[st.SelfID]; return e != nil && !e.Mounted })
	if err := world.Shutdown(10 * time.Second); err != nil {
		t.Fatal(err)
	}
	a.world = nil
	next, _, _, _ := testworld.Enter(t)
	if next.Inventory[[2]uint8{0, 13}].Type != 42 || next.Self.Mounted {
		t.Fatal("mount equipment or dismounted state not restored")
	}
	t.Log("equipped mount, self/remote shapes, animated model and dismount confirmed")
}

func TestVehicleModelsRealGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_VEHICLE") != "1" || os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original mount assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "mount models", Width: 512, Height: 512, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	assets := scene.NewAssets(arc)
	for race := uint8(0); race < 4; race++ {
		for index := uint8(0); index < 8; index++ {
			mount, err := assets.BuildVehicle(race, index)
			if err != nil {
				t.Fatalf("race %d MON %d: %v", race, index, err)
			}
			mount.World = render.Identity()
			mount.SetMoving(true)
			mount.Update(.1)
			primary := mount.Seat()
			if primary[1] < .1 || primary[1] > 4 {
				t.Fatal("invalid primary seat", race, index, primary)
			}
			budget := 8
			mount.PrepareEffects(.1, true, true, &budget)
			if code := gl.GetError(); code != gl.NO_ERROR {
				t.Fatalf("race %d MON %d GL error %#x", race, index, code)
			}
		}
	}
	for _, name := range []string{"ef_in_atrbu01", "ef_dmg_transf01", "run_horse-roar01", "run_pumav-roar01", "run_baslk-roar01"} {
		raw, err := arc.ReadFile("Sound/" + name + ".ogg")
		if err != nil {
			t.Fatal(err)
		}
		pcm, err := audio.DecodeOGG(raw)
		if err != nil || len(pcm) == 0 {
			t.Fatal("invalid mount audio", name, err)
		}
	}
	if len(assets.Missing) != 0 {
		t.Fatal("missing mount dependencies", assets.Missing)
	}
}
