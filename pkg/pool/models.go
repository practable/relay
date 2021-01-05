package pool

import (
	"sync"

	"github.com/timdrysdale/relay/pkg/permission"
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
	MinSession uint64
	MaxSession uint64
	Now        func() int64
}

type Description struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	DisplayInfo
}

type Activity struct {
	*sync.RWMutex
	Description
	ExpiresAt int64
	Streams   map[string]*Stream
	UI        []*UI
}

type UI struct {
	// URL with moustache {{key}} templating for stream connections
	Description
	URL             string `json:"url"`
	StreamsRequired []string
}

// Stream represents a data or video stream from a relay
// typically accessed via POST with bearer token
type Stream struct {
	*sync.RWMutex

	// For is the key in the UI's URL in which the client puts
	// the relay (wss) address and code after getting them
	// from the relay
	For string `json:"for,omitempty"`

	// URL of the relay access point for this stream
	URL string `json:"url"`

	// signed bearer token for accessing the stream
	// submit token in the header
	Token string `json:"token,omitempty"`

	// Verb is the HTTP method, typically post
	Verb string `json:"verb,omitempty"`

	// Permission is a prototype for the permission token that the booking system
	// generates and puts into the Token field
	Permission permission.Token `json:"permission,omitempty"`
}

type DisplayInfo struct {
	Short   string `json:"short,omitempty"`
	Long    string `json:"long,omitempty"`
	Further string `json:"further,omitempty"`
	Thumb   string `json:"thumb,omitempty"`
	Image   string `json:"image,omitempty"`
}
