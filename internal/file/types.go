package file

import (
	"regexp"
	"time"
)

type Comment struct {
	Msg  string
	Echo bool
}

type Error struct {
	string
}

type Send struct {
	Msg       string
	Delay     time.Duration
	Condition Condition
}

type Condition struct {
	AcceptPattern regexp.Regexp
	Count         int
	Timeout       time.Duration
}

type Filter struct {
	AcceptPatterns *map[string]regexp.Regexp
	DenyPatterns   *map[string]regexp.Regexp
}

type FilterAction struct {
	Verb    FilterVerb
	Pattern *regexp.Regexp
}

type FilterVerb int

const (
	Unknown FilterVerb = iota
	Accept
	Deny
	Reset
)
