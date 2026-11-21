package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

// One physical click on a mode icon must send exactly one command, however
// many frames the button is held.
func TestGMModeClickSendsOneCommand(t *testing.T) {
	a := inputApp()
	a.state.Self.IsGM = true
	var sent []string
	a.gmSend = func(text string) { sent = append(sent, text) }
	x, y := a.gmModes.Cell(1024, 768, 0)
	mx, my := x+10, y+10
	a.worldControls(&platform.Input{MouseX: mx, MouseY: my}, 1024, 768)
	a.worldControls(&platform.Input{MouseX: mx, MouseY: my, Pressed: true, MouseDown: true}, 1024, 768)
	for range 30 {
		a.worldControls(&platform.Input{MouseX: mx, MouseY: my, MouseDown: true}, 1024, 768)
	}
	a.worldControls(&platform.Input{MouseX: mx, MouseY: my, Clicked: true}, 1024, 768)
	for range 30 {
		a.worldControls(&platform.Input{MouseX: mx, MouseY: my}, 1024, 768)
	}
	if len(sent) != 1 || sent[0] != "/char on" {
		t.Fatal("one click sent", sent)
	}
}

// A second click before the server answered is ignored, so a burst of clicks
// cannot flood the world with visibility changes.
func TestGMModeBurstIsThrottledUntilTheAnswerWindowPasses(t *testing.T) {
	a := inputApp()
	a.state.Self.IsGM = true
	var sent []string
	a.gmSend = func(text string) { sent = append(sent, text) }
	for range 10 {
		a.toggleGMMode(1)
	}
	if len(sent) != 1 || sent[0] != "/attack on" {
		t.Fatal("burst", sent)
	}
	a.clock = 1.1
	a.state.GMImmune = true
	a.toggleGMMode(1)
	if len(sent) != 2 || sent[1] != "/attack off" {
		t.Fatal("after the window", sent)
	}
}
