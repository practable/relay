package hub

import "testing"

func TestMpsFromNs(t *testing.T) {

	if MpsFromNs(1e9) < 0.9999 {
		t.Error("Calculation wrong")
	}

	if MpsFromNs(1e9) > 1.0001 {
		t.Error("Calculation wrong")
	}

}
