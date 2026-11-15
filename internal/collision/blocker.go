package collision

import "math"

// BlockerBox constructs the explicit WLD box from 0x537840/0x5b2fd0.
// It must never be used as a substitute for a SMOD without collision geometry.
func BlockerBox(first, last Vec, yaw float64) [12]Triangle {
	d := last.Sub(first)
	points := [8]Vec{{0, 0, 0}, {0, d[1], 0}, {d[0], d[1], 0}, {d[0], 0, 0}, {d[0], d[1], d[2]}, {0, d[1], d[2]}, {0, 0, d[2]}, {d[0], 0, d[2]}}
	c, s := float64(float32(math.Cos(yaw))), float64(float32(math.Sin(yaw)))
	for i, p := range points {
		points[i] = Vec{float64(float32(p[0]*c + p[2]*s + first[0])), float64(float32(p[1] + first[1])), float64(float32(-p[0]*s + p[2]*c + first[2]))}
	}
	// Original index table at 0x70c3f0, not a guessed axis-aligned hull.
	faces := [12][3]int{{0, 1, 2}, {0, 2, 3}, {4, 5, 6}, {4, 6, 7}, {5, 1, 0}, {5, 0, 6}, {3, 2, 4}, {3, 4, 7}, {1, 5, 4}, {1, 4, 2}, {0, 3, 7}, {0, 7, 6}}
	var result [12]Triangle
	for i, f := range faces {
		result[i] = Triangle{points[f[0]], points[f[1]], points[f[2]]}
	}
	return result
}
