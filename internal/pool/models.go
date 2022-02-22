package pool

import (
	"sync"

	"github.com/timdrysdale/relay/pkg/permission"
)

// Store represents Activities stored according to Pool and Group
type Store struct {
	*sync.RWMutex `json:"-" yaml:"-"`

	// Groups represent non-exclusive combinations of pools
	Groups map[string]*Group `json:"groups"`

	// Pools maps all pools in the store
	Pools map[string]*Pool `json:"pools"`

	// Secret for generating tokens - assume one PoolStore per relay
	Secret []byte `json:"secret"`

	// How long to grant booking tokens for
	BookingTokenDuration int64 `json:"bookingTokenDuration"`

	// Now is a function for getting the time - useful for mocking in test
	Now func() int64 `json:"-" yaml:"-"`
}

// Group represents a group of Pools
type Group struct {
	*sync.RWMutex `json:"-" yaml:"-"`
	Description   `json:"description"`
	Pools         []*Pool `json:"pools"`
}

// Pool represents the booking status of the activities in a pool
// Note that each pool can have a different minSession / MaxSession duration
// but that users are limited to fixed maximum number of sessions they can book
// across the system to prevent users with access with more pools booking even more
// experiments simultaneously.
type Pool struct {
	*sync.RWMutex `json:"-" yaml:"-"`
	Description   `json:"description"`
	Activities    map[string]*Activity `json:"activities"`
	Available     map[string]int64     `json:"available"`
	InUse         map[string]int64     `json:"inUse"`
	MinSession    uint64               `json:"minSession"`
	MaxSession    uint64               `json:"maxSession"`
	Now           func() int64         `json:"-" yaml:"-"`
}

// Config represents a UI configuration file URL
type Config struct {
	URL string `json:"url"`
}

// Description represents an Activity Description
type Description struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	DisplayInfo
}

// Activity represents an individual activity that can be booked
type Activity struct {
	*sync.RWMutex `json:"-"`
	Config        Config `json:"config"`
	Description   `json:"description"`
	ExpiresAt     int64              `json:"exp"`
	Streams       map[string]*Stream `json:"streams"`
	UI            []*UI              `json:"ui"`
}

// UI represents a UI that can be used with an Activity
type UI struct {
	// URL with moustache {{key}} templating for stream connections
	Description     `json:"description"`
	URL             string   `json:"url"`
	StreamsRequired []string `json:"streamsRequired"`
}

// Stream represents a data or video stream from a relay
// typically accessed via POST with bearer token
type Stream struct {
	*sync.RWMutex `json:"-"`

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

// DisplayInfo represents information about an Activity that is displayed to users
type DisplayInfo struct {
	Short   string `json:"short,omitempty"`
	Long    string `json:"long,omitempty"`
	Further string `json:"further,omitempty"`
	Thumb   string `json:"thumb,omitempty"`
	Image   string `json:"image,omitempty"`
}
