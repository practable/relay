/*
   chanstats calculates statistics for bidirectional message channels
   Copyright (C) 2019 Timothy Drysdale <timothy.d.drysdale@gmail.com>

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as
   published by the Free Software Foundation, either version 3 of the
   License, or (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package chanstats

import (
	"time"

	"github.com/eclesh/welford"
)

// ChanStats represents recorded channel statistics
type ChanStats struct {
	ConnectedAt time.Time
	Rx          Messages
	Tx          Messages
}

// Messages represents statistics for messages
type Messages struct {
	Last  time.Time
	Bytes *welford.Stats
	Dt    *welford.Stats
}

// Report represents overally statistics for a topic
type Report struct {
	Connected string  `json:"connected"`
	Tx        Details `json:"tx"`
	Rx        Details `json:"rx"`
}

// Details represents detailed statistics
type Details struct {
	Last  string       `json:"last"` //how many seconds ago...
	Bytes WelfordStats `json:"bytes"`
	Dt    WelfordStats `json:"dt"`
}

// WelfordStats represents statistical values
type WelfordStats struct {
	Count    uint64  `json:"count"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Mean     float64 `json:"mean"`
	Stddev   float64 `json:"stddev"`
	Variance float64 `json:"variance"`
}

// New returns a pointer to new ChanStats struct with statistics initialised
func New() *ChanStats {
	c := &ChanStats{}
	c.ConnectedAt = time.Now() //expect user to update if appropriate
	c.Rx = Messages{Bytes: welford.New(), Dt: welford.New()}
	c.Tx = Messages{Bytes: welford.New(), Dt: welford.New()}
	return c
}

// NewReport represents a new report on channel statistics
func NewReport(s *ChanStats) *Report {
	r := &Report{
		Connected: s.ConnectedAt.String(),
		Rx:        *NewDetails(&s.Rx),
		Tx:        *NewDetails(&s.Tx),
	}
	return r
}

// NewDetails holds detailed information on channel statistics in one direction
func NewDetails(m *Messages) *Details {
	d := &Details{
		Last:  m.Last.String(),
		Bytes: *NewWelford(m.Bytes),
		Dt:    *NewWelford(m.Dt),
	}
	return d
}

// NewWelford initialises a new statistics structure
func NewWelford(w *welford.Stats) *WelfordStats {
	r := &WelfordStats{
		Count:    w.Count(),
		Min:      w.Min(),
		Max:      w.Max(),
		Mean:     w.Mean(),
		Stddev:   w.Stddev(),
		Variance: w.Variance(),
	}
	return r

}
