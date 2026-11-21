package app

import (
	"math"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestGMOnlyCameraZoomLimit(t *testing.T) {
	a := inputApp()
	v := a.wv
	v.world = &scene.World{}
	v.facing = render.V3{0, 0, 1}
	v.resetCamera()
	checkDistance := func(eye render.V3, want float64) {
		t.Helper()
		if got := float64(eye.Sub(v.cameraOrigin()).Len()); math.Abs(got-want) > .001 {
			t.Fatalf("camera distance %.4f, want %.4f", got, want)
		}
	}
	submit := func(command, want string) {
		t.Helper()
		a.chatInput, a.chatOpen = ui.TextField{Text: command}, true
		if !a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}) || a.chatOpen {
			t.Fatalf("%q was not handled locally", command)
		}
		if chat := a.state.Chat; chat[len(chat)-1].Text != want {
			t.Fatalf("%q returned %q, want %q", command, chat[len(chat)-1].Text, want)
		}
	}
	zoomOut := func() {
		for range 300 {
			v.cameraInput(&platform.Input{Wheel: -120}, 1.0/60, 0, 0)
		}
	}
	submit("/camlimit", "GM: access denied.")
	zoomOut()
	checkDistance(v.eye, zoomMax)
	a.state.Self.IsGM = true
	submit("/camlimit\t1000", "Usage: /camlimit")
	if v.extendedCameraZoom {
		t.Fatal("invalid command extended the camera range")
	}
	submit("/CAMLIMIT", "Camera zoom limit extended to 100.")
	checkDistance(v.eye, zoomMax)
	zoomOut()
	checkDistance(v.eye, 100)
	v.orbitCamera(0, .4)
	v.constrainCamera()
	v.updateRenderedCamera(.1)
	checkDistance(v.eye, 100)
	checkDistance(v.cameraEye, 100)
	submit("/camlimit", "Camera zoom limit restored to 13.")
	checkDistance(v.eye, zoomMax)
	checkDistance(v.cameraEye, zoomMax)
	zoomOut()
	checkDistance(v.eye, zoomMax)

	submit("/camlimit", "Camera zoom limit extended to 100.")
	zoomOut()
	v.updateRenderedCamera(.1)
	a.state.Self.IsGM = false
	a.updateWorldStep(&platform.Input{}, 1.0/60)
	if v.extendedCameraZoom {
		t.Fatal("camera range survived loss of GM status")
	}
	checkDistance(v.eye, zoomMax)
	checkDistance(v.cameraEye, zoomMax)
}

func TestSittingAndStandingPreserveWorldCamera(t *testing.T) {
	a := inputApp()
	v := a.wv
	v.world = &scene.World{}
	v.facing = render.V3{0, 0, 1}
	v.resetCamera()
	step := func() { a.updateWorldStep(&platform.Input{}, 1.0/60) }
	step()
	eye, target := v.cameraEye, v.cameraTarget
	for _, sitting := range []bool{true, false} {
		a.state.Self.Sitting = sitting
		step()
		if v.cameraEye != eye || v.cameraTarget != target {
			t.Fatalf("camera moved after sitting=%v: eye %v to %v, target %v to %v", sitting, eye, v.cameraEye, target, v.cameraTarget)
		}
	}
}

func TestMouseCameraWorksWithGroundClickMovementDisabled(t *testing.T) {
	a := inputApp()
	a.gameMenu.Settings.UseMouse = false
	a.gameMenu.Settings.MouseSensitivity = 0
	v := a.wv
	v.world = &scene.World{}
	v.facing = render.V3{0, 0, 1}
	v.resetCamera()
	eye := v.eye
	a.updateWorldStep(&platform.Input{RightDown: true, MouseLook: true, MouseDX: 120}, 1.0/60)
	if v.eye == eye || !v.mouselook {
		t.Fatal("right drag stopped controlling the camera")
	}
	distance := v.eye.Sub(v.cameraOrigin()).Len()
	a.updateWorldStep(&platform.Input{Wheel: 120}, 1.0/60)
	if v.eye.Sub(v.cameraOrigin()).Len() >= distance {
		t.Fatal("mouse wheel stopped controlling camera zoom")
	}
}
