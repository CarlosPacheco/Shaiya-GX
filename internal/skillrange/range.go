// Package skillrange shares spell launch/release reach with client prediction.
package skillrange

import "math"

const BodyAllowance float32 = 2.3
const ReleaseGrace float32 = 2.5

type Position [3]float32

func Reach(base, targetSize float32, completing bool) float32 {
	r := max(0, base) + max(0, targetSize) + BodyAllowance
	if completing {
		r += ReleaseGrace
	}
	return r
}
func Within(a, b Position, radius float32) bool {
	if radius < 0 || math.IsNaN(float64(radius)) || math.IsInf(float64(radius), 0) {
		return false
	}
	var d float64
	for i := range a {
		v := float64(a[i]) - float64(b[i])
		d += v * v
	}
	return d <= float64(radius)*float64(radius)
}
