package file

import (
	"fmt"
	"regexp"
	"time"
)

// Comment represents a comment in the playfile
// which can either be ignored, or echoed to the
// local logging file.
type Comment struct {
	Msg  string
	Echo bool
}

// Error represents an error in the parsing of the playfile
type Error struct {
	string
}

// Send is an instruction to send a message after a Delay
// and optionally wait for a condition to be met
// before moving to the next message to send
type Send struct {
	Msg       string
	Delay     time.Duration
	Condition Condition
}

// Wait represents an instruction to wait before
// proceeding to subsequent instructions,
// useful for testing mostly.
type Wait struct {
	Delay time.Duration
}

// Condition represents a condition for waiting for
// a number of responses meeting a pattern, or a
// maximum waiting time, whichever occurs first
type Condition struct {
	AcceptPattern regexp.Regexp
	Count         int
	Timeout       time.Duration
}

func (c *Condition) String() string {
	return fmt.Sprintf("<%s,%d,%s>",
		c.AcceptPattern.String(),
		c.Count,
		c.Timeout)
}

// Filter represents the setting of the logging Filter
// with a map of patterns to deny (DenyPatterns),
// and a map of patterns to accept (AcceptPatterns)
// The mapping is according to the original string
// used to construct the filter.
type Filter struct {
	AcceptPatterns *map[string]regexp.Regexp
	DenyPatterns   *map[string]regexp.Regexp
}

// FilterAction represents an action taken on the
// setting of the logging filter
type FilterAction struct {
	Verb    FilterVerb
	Pattern *regexp.Regexp
}

// FilterVerb represents names of actions that
// can be taken on settings of the logging filter
type FilterVerb int

const (
	Unknown FilterVerb = iota
	Accept
	Deny
	Reset
)

// Line represents content of a line received from the relay
// and the time it was received.
type Line struct {
	Time    time.Time
	Content string
}

type ConditionCheck struct {
	Condition Condition
	Satisfied chan struct{}
}
