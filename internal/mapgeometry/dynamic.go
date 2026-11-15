package mapgeometry

import (
	"math"
	"time"
)

var physicsEpoch = time.Now()

func PhysicsSeconds() float64                            { return time.Since(physicsEpoch).Seconds() }
func (g *Geometry) SetDynamicClock(clock func() float64) { g.clock = clock }
func (d Dynamic) Transform(elapsed float64) Matrix {
	a := norm(d.Axis)
	if a == ([3]float32{}) || d.Speed == 0 {
		return d.Base
	}
	t := math.Mod(elapsed*float64(d.Speed)*30, 2*math.Pi)
	c, s := float32(math.Cos(t)), float32(math.Sin(t))
	x, y, z := a[0], a[1], a[2]
	v := 1 - c
	rotation := Matrix{c + x*x*v, x*y*v + z*s, x*z*v - y*s, 0, y*x*v - z*s, c + y*y*v, y*z*v + x*s, 0, z*x*v + y*s, z*y*v - x*s, c + z*z*v, 0, 0, 0, 0, 1}
	var out Matrix
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				out[i*4+j] += rotation[i*4+k] * d.Base[k*4+j]
			}
		}
	}
	return out
}
