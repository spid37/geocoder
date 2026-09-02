package geocode_test

import "math"

func floatClose(a, b float64) bool {
	return math.Abs(a-b) < 1e-6
}
