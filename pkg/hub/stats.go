package hub

func MpsFromNs(ns float64) float64 {
	return 1 / (ns * 1e-9)
}
