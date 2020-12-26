package hub

import (
	"time"

	"github.com/eclesh/welford"
)

// Hub maintains the set of active clients and
// broadcasts messages to the clients.
// From gorilla/websocket chat
type Hub struct {
	// Registered clients.
	Clients map[string]map[*Client]bool

	// Inbound messages from the clients.
	Broadcast chan Message

	// Register requests from the clients.
	Register chan *Client

	// Unregister requests from clients.
	Unregister chan *Client

	Stats HubStats
}

type HubStats struct {
	Started  time.Time
	Last     time.Time
	Audience *welford.Stats
	Bytes    *welford.Stats
	Latency  *welford.Stats
	Dt       *welford.Stats
}

// Stats that we report externally
type HubReport struct {
	Started  string       `json:"started"`
	Last     string       `json:"last"`
	Audience WelfordStats `json:"audience"`
	Bytes    WelfordStats `json:"bytes"`
	Latency  WelfordStats `json:"latency"`
	Dt       WelfordStats `json:"dt"`
}

type WelfordStats struct {
	Count    uint64  `json:"count"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Mean     float64 `json:"mean"`
	Stddev   float64 `json:"stddev"`
	Variance float64 `json:"variance"`
}

// messages will be wrapped in this struct for muxing
type Message struct {
	Data   []byte //text data are converted to/from bytes as needed
	Sender Client
	Sent   time.Time //when sent
	Type   int
}

// Client is a middleperson between the hub and whatever is sending/receiving messages on it
type Client struct {
	Hub   *Hub
	Name  string       //for filtering who to send messages to
	Send  chan Message // for outbound messages to client
	Stats *ClientStats
	Topic string // message broadcast scope is restricted to a single topic
	Done  chan struct{}
}

// Stats that we keep internally
type ClientStats struct {
	ConnectedAt time.Time
	Rx          *Frames
	Tx          *Frames
}

type Frames struct {
	Last time.Time
	Size *welford.Stats
	Dt   *welford.Stats
}

// Stats that we report externally
type ClientReport struct {
	Topic     string     `json:"topic"`
	Connected string     `json:"connected"`
	Stats     ClientRxTx `json:"stats"`
}

type ClientRxTx struct {
	Tx ChannelStats `json:"tx"`
	Rx ChannelStats `json:"rx"`
}

type ChannelStats struct {
	Last  string  `json:"last"` //how many seconds ago...
	Bytes float64 `json:"bytes"`
	Dt    float64 `json:"dt"` //Messages per second
}
