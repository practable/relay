package hub

// MpsFromNs returns frequency of event occurring every ns nanoseconds
func MpsFromNs(ns float64) float64 {
	return 1 / (ns * 1e-9)
}
