package mathx

import "math"

func Round(v float64, precision int) float64 {
	if precision == 0 {
		return math.Round(v)
	}

	p := math.Pow10(precision)
	if precision < 0 {
		return math.Floor(v*p+0.5) + math.Pow10(-precision)
	}

	return math.Floor(v*p+0.5) / p
}
