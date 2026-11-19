package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestWeatherContinuesFallingAfterCameraMoves(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind uint8
		want int
		fall float32
	}{
		{"rain", proto.WeatherRain, 25, -50},
		{"snow", proto.WeatherSnow, 8, -2.5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var view weatherView
			view.update(tc.kind, .100, render.V3{0, 15, 20})
			if len(view.drops) != tc.want {
				t.Fatalf("first 100ms emitted %d drops, want %d", len(view.drops), tc.want)
			}
			before := view.drops[0]
			view.update(tc.kind, .100, render.V3{100, 15, 20})
			after := view.drops[0]
			if after.position != before.position || after.age <= before.age || after.velocity[1] != tc.fall {
				t.Fatalf("particle birth or fall changed when camera moved: %+v → %+v", before, after)
			}
			for range 80 {
				view.update(0, .100, render.V3{})
			}
			if view.blend != 0 || len(view.drops) != 0 {
				t.Fatalf("clear weather kept storm backdrop or particles")
			}
		})
	}
}

func TestRainFogBecomesNeutralGray(t *testing.T) {
	view := weatherView{kind: proto.WeatherRain, blend: 1}
	color, _, _ := view.fog(render.V3{.15, .4, .9}, 100, 200)
	if color[0] != color[1] || color[1] != color[2] {
		t.Fatalf("rain fog retained a color cast: %v", color)
	}
}
