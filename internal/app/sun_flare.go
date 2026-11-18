package app

import (
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/go-gl/gl/v4.5-core/gl"
)

type sunFlare struct {
	x, y       float32
	visibility float32
	strength   float32
}

// PS107 Game.exe 0x538cc0, 0x539020: fixed sky position and seven additive
// quads. The camera fades the effect through world collision at 4 units/s.
var sunPosition = render.V3{0, 5000, 10000}

var sunFlareLayers = [...]struct {
	texture     string
	along, size float32
	color       [3]float32
}{
	{"Lensflare/flare4.jpg", 0, .6, [3]float32{.7, .4, .4}},
	{"Lensflare/flare0.jpg", 0, .25, [3]float32{1, 1, 1}},
	{"Lensflare/flare1.jpg", 0, .2, [3]float32{.3, .1, 0}},
	{"Lensflare/flare5.jpg", .4, .2, [3]float32{.07, .07, .1}},
	{"Lensflare/flare5.jpg", .32, .05, [3]float32{.07, .07, .1}},
	{"Lensflare/flare5.jpg", .46, .1, [3]float32{.07, .07, .1}},
	{"Lensflare/flare5.jpg", .8, .06, [3]float32{.3, .2, .13}},
}

func projectSun(view, projection render.M4) (x, y float32, inView bool) {
	// The original removes X/Z view translation and scales Y by .015.
	view[12], view[13], view[14] = 0, view[13]*.015, 0
	z := sunPosition[1]*view[6] + sunPosition[2]*view[10]
	if z < .5 {
		return 0, 0, false
	}
	p := view.Mul(projection)
	w := sunPosition[1]*p[7] + sunPosition[2]*p[11] + p[15]
	x = (sunPosition[1]*p[4] + sunPosition[2]*p[8] + p[12]) / w
	y = (sunPosition[1]*p[5] + sunPosition[2]*p[9] + p[13]) / w
	return x, y, x >= -1.2 && x <= 1.2 && y >= -1.2 && y <= 1.2
}

func (f *sunFlare) update(world *scene.World, eye render.V3, view, projection render.M4, far, dt float32) {
	f.strength = 0
	if world.Sky == nil || world.WLD == nil || world.WLD.Dungeon || eye[1] <= 0 {
		f.visibility = 0
		return
	}
	x, y, inView := projectSun(view, projection)
	if !inView {
		return
	}
	f.x, f.y = x, y
	if world.SkyVisible(eye, sunPosition, far) {
		f.visibility = min(1, f.visibility+dt*4)
	} else {
		f.visibility = max(0, f.visibility-dt*4)
	}
	edge := max(0, x-1) + max(0, -x-1) + max(0, y-1) + max(0, -y-1)
	f.strength = max(0, f.visibility-5*edge)
}

func (a *App) drawSunFlare(width, height int) {
	if a.wv == nil || a.assets == nil || a.wv.sunFlare.strength <= 0 || width <= 0 || height <= 0 {
		return
	}
	f := a.wv.sunFlare
	halfW, halfH := float32(width)*.5, float32(height)*.5
	verticalScale := (math.Pi / 2) / projectionAspect(width, height)
	gl.BlendFunc(gl.ONE, gl.ONE)
	for _, layer := range sunFlareLayers {
		tex := a.assets.Texture(layer.texture)
		if tex == nil {
			continue
		}
		x := (1 + f.x*(1-layer.along)) * halfW
		y := (1 - f.y*(1-layer.along)) * halfH
		dx, dy := layer.size*halfW, layer.size*verticalScale*halfH
		c := layer.color
		a.sprites.Draw(tex, x-dx, y-dy, dx*2, dy*2, 0, 0, 1, 1,
			[4]float32{c[0] * f.strength, c[1] * f.strength, c[2] * f.strength, 1})
	}
	a.sprites.Flush()
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
}
