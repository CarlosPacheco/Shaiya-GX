// Package skillrange shares spell launch/release reach with client prediction.
package skillrange

import "math"

const BodyAllowance float32 = 2.3

type Position [3]float32

func Reach(base, targetSize float32) float32 {
	return max(0, base) + max(0, targetSize) + BodyAllowance
}
func Finite(p Position) bool {
	for _, v := range p {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return false
		}
	}
	return true
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
