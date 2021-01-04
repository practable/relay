package pool

import (
	"sync"
)

type PoolStore struct {
	*sync.RWMutex

	// Groups represent non-exclusive combinations of pools
	Groups map[string]*Group

	// Pools maps all pools in the store
	Pools map[string]*Pool

	// Secret for generating tokens - assume one PoolStore per relay
	Secret []byte

	// How long to grant booking tokens for
	BookingTokenDuration int64

	// Now is a function for getting the time - useful for mocking in test
	Now func() int64
}

type Group struct {
	*sync.RWMutex
	Description
	Pools []*Pool
}

type Pool struct {
	*sync.RWMutex
	Description
	Activities map[string]*Activity
	Available  map[string]int64
	InUse      map[string]int64
	Now        func() int64
}

type Description struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	DisplayInfo
}

type Permission struct {
	Topic          string
	Audience       string
	ConnectionType string
}

type Activity struct {
	*sync.RWMutex
	Description
	ExpiresAt  int64
	Streams    map[string]*Stream
	Permission Permission
}

// Stream represents a data or video stream from a relay
// typically accessed via POST with bearer token
type Stream struct {
	For   string `json:"for,omitempty"`
	URL   string `json:"url"`
	Token string `json:"token,omitempty"`
	Verb  string `json:"verb,omitempty"`
}

type DisplayInfo struct {
	Short   string `json:"short,omitempty"`
	Long    string `json:"long,omitempty"`
	Further string `json:"further,omitempty"`
	Thumb   string `json:"thumb,omitempty"`
	Image   string `json:"image,omitempty"`
}
