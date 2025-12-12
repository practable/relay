package file

import (
	"time"
)

// Error represents an error in the parsing of the playfile
type Error struct {
	string
}

// Line represents content of a line received from the relay
// and the time it was received.
type Line struct {
	Time    time.Time
	Content string
}
