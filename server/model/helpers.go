package model

func SafeDiv(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b) * 100
}
