package render

import "github.com/go-gl/gl/v4.5-core/gl"

var uiPixelScale = 1.0

// SetUIScale tells InViewport how many framebuffer pixels one UI unit is.
func SetUIScale(k float64) { uiPixelScale = k }

// ClearBackground replaces only the colour buffer and preserves the caller's
// clear colour; the world depth and later UI passes keep their existing state.
func ClearBackground(color V3) {
	clear := [4]float32{color[0], color[1], color[2], 1}
	gl.ClearBufferfv(gl.COLOR, 0, &clear[0])
}

// InViewport draws a 3D UI region without clearing its background or changing
// the enclosing viewport/scissor. Coordinates are framebuffer pixels from top left.
//
// When the UI is drawn in a virtual window (SetUIScale < 1), x, y, w, h and
// screenHeight are virtual units and are converted to framebuffer pixels.
func InViewport(x, y, w, h, screenHeight int, draw func()) {
	if k := uiPixelScale; k != 1 {
		x0, y0 := int(float64(x)*k+.5), int(float64(y)*k+.5)
		x1, y1 := int(float64(x+w)*k+.5), int(float64(y+h)*k+.5)
		x, y, w, h, screenHeight = x0, y0, max(1, x1-x0), max(1, y1-y0), int(float64(screenHeight)*k+.5)
	}
	var viewport, scissor [4]int32
	var clearDepth float64
	gl.GetIntegerv(gl.VIEWPORT, &viewport[0])
	gl.GetIntegerv(gl.SCISSOR_BOX, &scissor[0])
	gl.GetDoublev(gl.DEPTH_CLEAR_VALUE, &clearDepth)
	enabled := gl.IsEnabled(gl.SCISSOR_TEST)
	gl.Viewport(int32(x), int32(screenHeight-y-h), int32(w), int32(h))
	gl.Scissor(int32(x), int32(screenHeight-y-h), int32(w), int32(h))
	gl.Enable(gl.SCISSOR_TEST)
	gl.DepthMask(true)
	// UI previews use ordinary depth independently of the world's reverse Z.
	gl.ClearDepth(1)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	draw()
	gl.ClearDepth(clearDepth)
	gl.Viewport(viewport[0], viewport[1], viewport[2], viewport[3])
	gl.Scissor(scissor[0], scissor[1], scissor[2], scissor[3])
	if !enabled {
		gl.Disable(gl.SCISSOR_TEST)
	}
}
