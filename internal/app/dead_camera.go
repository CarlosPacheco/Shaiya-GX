package app

import (
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

// deadCameraInput keeps only what orbits and zooms the camera. A dead
// character cannot move, turn, act or target, but the view stays free.
func (a *App) deadCameraInput(in *platform.Input) platform.Input {
	camera := platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true,
		Wheel: in.Wheel, MiddlePressed: in.MiddlePressed, MouseLook: in.MouseLook, MouseDX: in.MouseDX, MouseDY: in.MouseDY}
	if a.wv == nil || !a.wv.uiRight {
		camera.RightDown, camera.RightPressed = in.RightDown, in.RightPressed
	}
	// Arrow keys and +/- are camera controls; letters would move the character.
	camera.Down = strings.Map(func(r rune) rune {
		if strings.ContainsRune("<>^v+-", r) {
			return r
		}
		return -1
	}, in.Down)
	for _, k := range in.Keys {
		if k == platform.KeyF9 || k == platform.KeyF10 || k == platform.KeyHome {
			camera.Keys = append(camera.Keys, k)
		}
	}
	return camera
}
