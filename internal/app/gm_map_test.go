package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"testing"
)

func TestMapClickVersusDrag(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mid     platform.Input
		outside bool
		want    bool
	}{
		{name: "click", want: true},
		{name: "drag", mid: platform.Input{MouseX: 20, MouseY: 20, MouseDown: true}},
		{name: "wheel", mid: platform.Input{Wheel: 120, MouseDown: true}},
		{name: "release outside", outside: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var c mapClick
			c.update(&platform.Input{Pressed: true, MouseDown: true}, true)
			if tc.mid.MouseDown {
				c.update(&tc.mid, true)
			}
			if got := c.update(&platform.Input{Clicked: true}, !tc.outside); got != tc.want {
				t.Fatal(got)
			}
			if c.update(&platform.Input{Clicked: true}, true) {
				t.Fatal("release repeated")
			}
		})
	}
}
