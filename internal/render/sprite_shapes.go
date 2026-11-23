package render

import (
	"math"

	"github.com/go-gl/gl/v4.5-core/gl"
)

// DrawCircle clips one minimap texture to a screen-space circle, like the
// Phoenix MinimapHud MAP_SHADER. UVs remain in the source map's coordinate space.
func (s *Sprites) DrawCircle(t *Texture, x, y, diameter, u0, v0, u1, v1 float32, tint [4]float32) {
	s.Flush()
	radius := diameter / 2
	gl.Uniform3f(s.uCircle, x+radius, y+radius, radius)
	s.Draw(t, x, y, diameter, diameter, u0, v0, u1, v1, tint)
	s.Flush()
	gl.Uniform3f(s.uCircle, 0, 0, 0)
}

// DrawCircleClipped is DrawCircle for a window that may extend past the image:
// only the part of the quad that samples inside [0,1] is drawn, still masked by
// the circle.
func (s *Sprites) DrawCircleClipped(t *Texture, x, y, diameter, u0, v0, u1, v1 float32, tint [4]float32) {
	if u1 <= u0 || v1 <= v0 {
		return
	}
	cu0, cu1 := max(u0, 0), min(u1, 1)
	cv0, cv1 := max(v0, 0), min(v1, 1)
	if cu1 <= cu0 || cv1 <= cv0 {
		return
	}
	w, h := diameter/(u1-u0), diameter/(v1-v0) // pixels per unit of u and v
	s.Flush()
	radius := diameter / 2
	gl.Uniform3f(s.uCircle, x+radius, y+radius, radius)
	s.Draw(t, x+(cu0-u0)*w, y+(cv0-v0)*h, (cu1-cu0)*w, (cv1-cv0)*h, cu0, cv0, cu1, cv1, tint)
	s.Flush()
	gl.Uniform3f(s.uCircle, 0, 0, 0)
}

// DrawRotated rotates an ordinary atlas quad clockwise around its screen centre.
func (s *Sprites) DrawRotated(t *Texture, cx, cy, w, h, angle, u0, v0, u1, v1 float32, tint [4]float32) {
	s.Draw(t, cx-w/2, cy-h/2, w, h, u0, v0, u1, v1, tint)
	sine, cosine := math.Sincos(float64(angle))
	sn, cs := float32(sine), float32(cosine)
	for i := len(s.verts) - 6; i < len(s.verts); i++ {
		v := &s.verts[i]
		x, y := v.x-cx, v.y-cy
		v.x, v.y = cx+x*cs-y*sn, cy+x*sn+y*cs
	}
}
