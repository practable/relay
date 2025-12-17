package file

import (
	"time"
)

// Line represents content of a line received from the relay
// and the time it was received.
type Line struct {
	Time    time.Time
	Content string
}
