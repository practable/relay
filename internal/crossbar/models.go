package crossbar

import (
	"sync"
	"time"

	"github.com/eclesh/welford"
	"github.com/gorilla/websocket"
	"github.com/practable/relay/internal/deny"
	"github.com/practable/relay/internal/ttlcode"
)

// Config represents configuration options for a crossbar instance
// Use this struct to pass configuration as argument during testing
type Config struct {

	// Listen is the listening port
	Listen int

	// Audience must match the host in token
	Audience string

	// Secret is used to validating statsTokens
	Secret string

	// ExchangeCode swaps a code for the associated Token
	CodeStore *ttlcode.CodeStore

	//DenyStore holds deny-listed bookingIDs
	DenyStore *deny.Store
}

// NewDefaultConfig returns a pointer to a Config struct with default parameters
func NewDefaultConfig() *Config {
	c := &Config{}
	c.Listen = 3000
	c.CodeStore = ttlcode.NewDefaultCodeStore()
	return c
}

// WithListen specified which (int) port to listen on
func (c *Config) WithListen(listen int) *Config {
	c.Listen = listen
	return c
}

// WithAudience specificies the audience for the tokens
func (c *Config) WithAudience(audience string) *Config {
	c.Audience = audience
	return c
}

// WithCodeStoreTTL specifies the lifetime for the codestore
func (c *Config) WithCodeStoreTTL(ttl int64) *Config {
	c.CodeStore = ttlcode.NewDefaultCodeStore().
		WithTTL(ttl)
	return c
}

// Client is a middleperson between the websocket connection and the hub.
type Client struct {

	// bookingID from the token
	bookingID string

	// hub closes this channel if connection is curtailed
	denied chan struct{}

	expiresAt int64

	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan message

	// string representing the path the client connected to
	topic string

	audience string

	stats *Stats

	name string

	userAgent string

	remoteAddr string

	// existence of scopes to read, write
	canRead, canWrite bool
}

//RxTx represents statistics for both receive and transmit
type RxTx struct {
	Tx ReportStats `json:"tx"`
	Rx ReportStats `json:"rx"`
}

// ReportStats represents statistics about what has been sent/received
type ReportStats struct {
	Last string `json:"last"` //how many seconds ago...

	Size float64 `json:"size"`

	Fps float64 `json:"fps"`
}

// ClientReport represents information about a client's connection, permissions, and statistics
type ClientReport struct {
	Topic string `json:"topic"`

	CanRead bool `json:"canRead"`

	CanWrite bool `json:"canWrite"`

	Connected string `json:"connected"`

	RemoteAddr string `json:"remoteAddr"`

	UserAgent string `json:"userAgent"`

	Stats RxTx `json:"stats"`
}

// StatsCommand represents a command in string form
type StatsCommand struct {
	Command string `json:"cmd"`
}

// Stats represents statistics for a connection
type Stats struct {
	connectedAt time.Time

	rx *Frames

	tx *Frames
}

// Frames represents statistics on (video) frames sent over a connection
type Frames struct {
	last time.Time

	size *welford.Stats

	ns *welford.Stats

	mu *sync.RWMutex
}

// messages will be wrapped in this struct for muxing
type message struct {
	sender Client
	mt     int
	data   []byte //text data are converted to/from bytes as needed
}

// TODO - remove unused types below this line (some still in use)

type clientDetails struct {
	name         string
	topic        string
	messagesChan chan message
}

// requests to add or delete subscribers are represented by this struct
type clientAction struct {
	action clientActionType
	client clientDetails
}

// userActionType represents the type of of action requested
type clientActionType int

// clientActionType constants
const (
	clientAdd clientActionType = iota
	clientDelete
)

type topicDirectory struct {
	sync.Mutex
	directory map[string][]clientDetails
}
