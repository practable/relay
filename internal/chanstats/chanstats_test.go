package chanstats

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"math/big"
)

var epsilon, tolerance *big.Float

var chunk1 = string(`"bytes":{"count":4,"min":1000,"max":4000,"mean":2500,"stddev":1290.9944487358057,"variance":1666666.6666666667},"dt":{"count":4,"min":0.019,"max":0.021,"mean":0.02,"stddev":0.0008164965809277268,"variance":6.666666666666679e-7}`)

var chunk2 = string(`"bytes":{"count":4,"min":100,"max":400,"mean":250,"stddev":129.09944487358058,"variance":16666.666666666668},"dt":{"count":4,"min":0.009,"max":0.011,"mean":0.009999999999999998,"stddev":0.0008164965809277256,"variance":6.666666666666661e-7}`)

func init() {
	epsilon = big.NewFloat(math.Nextafter(1.0, 2.0) - 1.0)
	tolerance = big.NewFloat(1.01) //one percent
}

func TestInitialise(t *testing.T) {

	s := New()

	if time.Since(s.ConnectedAt) > time.Millisecond {
		t.Error("ConnectedAt not initialised")
	}
	if s.Rx.Bytes.Count() != 0 {
		t.Error("Rx.Bytes not initialised")
	}
	if s.Rx.Dt.Count() != 0 {
		t.Error("Rx.Dt not initialised")
	}
	if s.Tx.Bytes.Count() != 0 {
		t.Error("Tx.Bytes not initialised")
	}
	if s.Tx.Dt.Count() != 0 {
		t.Error("Tx.Dt not initialised")
	}

}

func TestUpdateStats(t *testing.T) {

	var wanted, got *big.Float

	s := New()

	loadStats(s)

	if s.Rx.Bytes.Count() != 4 {
		t.Error("Rx.Bytes incorrect count", s.Rx.Bytes.Count())
	}
	if s.Tx.Bytes.Count() != 4 {
		t.Error("Tx.Bytes incorrect count", s.Tx.Bytes.Count())
	}
	if s.Rx.Bytes.Mean() != 250 {
		t.Error("Rx.Bytes incorrect mean", s.Rx.Bytes.Mean())
	}
	if s.Tx.Bytes.Mean() != 2500 {
		t.Error("Tx.Bytes incorrect mean", s.Tx.Bytes.Mean())
	}
	if s.Rx.Dt.Count() != 4 {
		t.Error("Rx.Dt incorrect count", s.Rx.Dt.Count())
	}
	if s.Tx.Dt.Count() != 4 {
		t.Error("Tx.Dt incorrect count", s.Tx.Dt.Count())
	}

	wanted = big.NewFloat(0.01)
	got = big.NewFloat(s.Rx.Dt.Mean())
	if wanted.Cmp(big.NewFloat(0).Mul(got, tolerance)) > 0 {
		t.Errorf("Rx.Dt incorrect mean wanted %f, got %f\n", wanted, got)
	}
	if wanted.Cmp(big.NewFloat(0).Quo(got, tolerance)) < 0 {
		t.Errorf("Rx.Dt incorrect mean wanted %f, got %f\n", wanted, got)
	}

	wanted = big.NewFloat(0.02)
	got = big.NewFloat(s.Tx.Dt.Mean())
	if wanted.Cmp(big.NewFloat(0).Mul(got, tolerance)) > 0 {
		t.Errorf("Tx.Dt incorrect mean wanted %f, got %f\n", wanted, got)
	}
	if wanted.Cmp(big.NewFloat(0).Quo(got, tolerance)) < 0 {
		t.Errorf("Tx.Dt incorrect mean wanted %f, got %f\n", wanted, got)
	}

}
func TestReport(t *testing.T) {

	var wanted, got *big.Float

	s := New()

	loadStats(s)

	r := NewReport(s)

	if r.Rx.Bytes.Max != 400 {
		t.Error("r.Rx.Bytes.Max reported wrong")
	}

	wanted = big.NewFloat(s.Tx.Dt.Stddev())
	got = big.NewFloat(r.Tx.Dt.Stddev)

	if wanted.Cmp(big.NewFloat(0).Mul(got, tolerance)) > 0 {
		t.Errorf("Tx.Dt incorrect stddev  wanted %f, got %f\n", wanted, got)
	}
	if wanted.Cmp(big.NewFloat(0).Quo(got, tolerance)) < 0 {
		t.Errorf("Tx.Dt incorrect stddev  wanted %f, got %f\n", wanted, got)
	}
}

func TestMarshalJSON(t *testing.T) {
	s := New()
	loadStats(s)
	r := NewReport(s)
	j, err := json.Marshal(r)
	if err != nil {
		t.Error("Error marshalling report into JSON", err)
	}
	if !strings.Contains(string(j), chunk1) {
		t.Error("JSON not marshalled correctly.")
	}
	if !strings.Contains(string(j), chunk2) {
		t.Error("JSON not marshalled correctly.")
	}
}

func loadStats(s *ChanStats) {

	rxSizes := []float64{100, 200, 300, 400}
	txSizes := []float64{1000, 2000, 3000, 4000}
	rxDt := []float64{0.01, 0.011, 0.009, 0.01}
	txDt := []float64{0.02, 0.021, 0.019, 0.02}

	for i := 0; i < len(rxSizes); i++ {
		s.Rx.Bytes.Add(rxSizes[i])
		s.Tx.Bytes.Add(txSizes[i])
		s.Rx.Dt.Add(rxDt[i])
		s.Tx.Dt.Add(txDt[i])
	}

}
