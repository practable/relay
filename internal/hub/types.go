package hub

import (
	"time"
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
}

// Message represents a message that is wrapped and ready for multiplexing
// unwrapped messages cannot be multiplexed
type Message struct {
	Data   []byte //text data are converted to/from bytes as needed
	Sender Client
	Sent   time.Time //when sent
	Type   int
}

// Client is a middleperson between the hub and whatever is sending/receiving messages on it
type Client struct {
	Hub         *Hub
	Name        string       //for filtering who to send messages to
	Send        chan Message // for outbound messages to client
	Topic       string       // message broadcast scope is restricted to a single topic
	Done        chan struct{}
	ConnectedAt time.Time
}

// ClientReport represents statistics that we report externally
type ClientReport struct {
	Topic       string `json:"topic"`
	ConnectedAt string `json:"connected"`
}
