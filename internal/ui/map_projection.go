package ui

import "math"

// MapProjection is the world-space width/height covered by the entire image.
// Fields use their WLD size; dungeons use image pixels / native CFG scales.
type MapProjection struct{ Width, Height float32 }

func (p MapProjection) Position(x, z float32) (u, v float32, ok bool) {
	if p.Width <= 0 || p.Height <= 0 {
		return
	}
	u, v = x/p.Width, 1-z/p.Height
	return u, v, u >= 0 && u <= 1 && v >= 0 && v <= 1
}
func (p MapProjection) Coordinates(view UV, fx, fy float32) (x, z float32, ok bool) {
	u, z, ok := MapCoordinates(view, fx, fy, 1)
	if !ok || p.Width <= 0 || p.Height <= 0 {
		return 0, 0, false
	}
	return u * p.Width, z * p.Height, true
}
func (p MapProjection) Window(x, z, diameter float32) UV {
	u, v, _ := p.Position(x, z)
	if p.Width <= 0 || p.Height <= 0 {
		return MapWindow(.5, .5, 1)
	}
	return MapWindowXY(u, v, diameter/p.Width, diameter/p.Height)
}

// MinimapWindow is the native small-map sampling (EP5.4 0x4c7860): half the
// map texture spans the 256-pixel backing quad at zoom 1, the player is always
// the pixel (MinimapAnchorX, MinimapAnchorY) of that quad whatever the map
// edge, and its position is truncated to 1/512 of the map. The window is not
// clamped: past the map edge it simply shows nothing.
func (p MapProjection) MinimapWindow(x, z, zoom float32) UV {
	if p.Width <= 0 || p.Height <= 0 {
		return MapWindow(.5, .5, 1)
	}
	u := float32(math.Trunc(float64(x/p.Width*512))) / 512
	v := 1 - float32(math.Trunc(float64(z/p.Height*512)))/512
	s := zoom / 512 // map fraction per screen pixel
	return UV{
		u + (MinimapCircleEdge-MinimapAnchorX)*s, v + (MinimapCircleEdge-MinimapAnchorY)*s,
		u + (MinimapCircleEdge+MinimapImageSize-MinimapAnchorX)*s, v + (MinimapCircleEdge+MinimapImageSize-MinimapAnchorY)*s,
	}
}

// MapWindowXY preserves independent authored scales on the two image axes.
func MapWindowXY(u, v, spanU, spanV float32) UV {
	spanU, spanV = min(1, max(.001, spanU)), min(1, max(.001, spanV))
	hu, hv := spanU/2, spanV/2
	u, v = min(1-hu, max(hu, u)), min(1-hv, max(hv, v))
	return UV{u - hu, v - hv, u + hu, v + hv}
}
