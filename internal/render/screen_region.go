package render

import "math"

// ScreenRegion uses normalized lower-left framebuffer coordinates.
type ScreenRegion struct{ X0, Y0, X1, Y1 float32 }

var FullScreenRegion = ScreenRegion{0, 0, 1, 1}

func (r ScreenRegion) Expand(x, y float32) ScreenRegion {
	return ScreenRegion{max(0, r.X0-x), max(0, r.Y0-y), min(1, r.X1+x), min(1, r.Y1+y)}
}

func (r ScreenRegion) Pixels(width, height int32) (x, y, w, h int32) {
	x, y = int32(math.Floor(float64(r.X0*float32(width)))), int32(math.Floor(float64(r.Y0*float32(height))))
	right, top := int32(math.Ceil(float64(r.X1*float32(width)))), int32(math.Ceil(float64(r.Y1*float32(height))))
	return x, y, max(right-x, 0), max(top-y, 0)
}

// CropProjection narrows culling only. The raster projection and viewport
// stay unchanged so reflection sampling keeps its original coordinates.
func (r ScreenRegion) CropProjection(p M4) M4 {
	w, h := r.X1-r.X0, r.Y1-r.Y0
	if w <= 0 || h <= 0 || r == FullScreenRegion {
		return p
	}
	crop := Identity()
	crop[0], crop[5] = 1/w, 1/h
	crop[12], crop[13] = (1-r.X0-r.X1)/w, (1-r.Y0-r.Y1)/h
	return p.Mul(crop)
}
