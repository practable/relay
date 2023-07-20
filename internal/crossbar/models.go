package crossbar

import (
	"sync"
	"time"

	"github.com/eclesh/welford"
	"github.com/gorilla/websocket"
	"github.com/practable/relay/internal/deny"
	"github.com/practable/relay/internal/ttlcode"
	log "github.com/sirupsen/logrus"
)

// Config represents configuration options for a crossbar instance
// Use this struct to pass configuration as argument during testing
type Config struct {

	// Audience must match the host in token
	Audience string

	//BufferSize sets the buffer size for client communications channels
	BufferSize int64

	// ExchangeCode swaps a code for the associated Token
	CodeStore *ttlcode.CodeStore

	//DenyStore holds deny-listed bookingIDs
	DenyStore *deny.Store

	// Listen is the listening port
	Listen int

	// Secret is used to validating statsTokens
	Secret string

	//StatsEvery sets how often stats are reported
	StatsEvery time.Duration
}

// NewDefaultConfig returns a pointer to a Config struct with default parameters
func NewDefaultConfig() *Config {
	c := &Config{}
	c.Listen = 3000
	c.CodeStore = ttlcode.NewDefaultCodeStore()
	c.BufferSize = 128
	c.StatsEvery = time.Duration(5 * time.Second)
	log.WithFields(log.Fields{"BufferSize": c.BufferSize, "listen": c.Listen, "ttl": c.CodeStore.GetTTL()}).Info("crossbar default config")
	return c
}

// WithBufferSize specifies the client channel buffer size
func (c *Config) WithBufferSize(n int64) *Config {
	if n > 0 && n <= 512 {
		c.BufferSize = n
	} else {
		log.WithFields(log.Fields{"requested": n, "actual": c.BufferSize}).Error("BufferSize must be between 1 - 512 (128 recommended)")
	}
	log.WithFields(log.Fields{"BufferSize": n}).Info("crossbar buffer size set")
	return c
}

// WithStatsEvery specifies how often to send stats
func (c *Config) WithStatsEvery(statsEvery time.Duration) *Config {
	c.StatsEvery = statsEvery
	log.WithFields(log.Fields{"statsEvery": statsEvery}).Info("crossbar statsEvery set")
	return c
}

// WithListen specified which (int) port to listen on
func (c *Config) WithListen(listen int) *Config {
	c.Listen = listen
	log.WithFields(log.Fields{"listen": listen}).Info("crossbar listening port set")
	return c
}

// WithAudience specificies the audience for the tokens
func (c *Config) WithAudience(audience string) *Config {
	c.Audience = audience
	log.WithFields(log.Fields{"audience": audience}).Info("crossbar audience set")
	return c
}

// WithCodeStoreTTL specifies the lifetime for the codestore
func (c *Config) WithCodeStoreTTL(ttl int64) *Config {
	c.CodeStore = ttlcode.NewDefaultCodeStore().
		WithTTL(ttl)
	log.WithFields(log.Fields{"ttl": ttl}).Info("crossbar ttl set")
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
	CanRead bool `json:"canRead"`

	CanWrite bool `json:"canWrite"`

	Connected string `json:"connected"`

	ExpiresAt string `json:"expiresAt"`

	RemoteAddr string `json:"remoteAddr"`

	Stats RxTx `json:"stats"`

	Topic string `json:"topic"`

	UserAgent string `json:"userAgent"`
}

// StatsCommand represents a command in string form
type StatsCommand struct {
	Command string `json:"cmd"`
}

// Stats represents statistics for a connection
type Stats struct {
	connectedAt time.Time

	expiresAt time.Time

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

// userActionType represents the type of of action requested
//type clientActionType int

type topicDirectory struct {
	sync.Mutex
	directory map[string][]clientDetails
}
