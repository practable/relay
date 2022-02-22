package rwc

import (
	"context"

	"github.com/timdrysdale/relay/pkg/agg"
	"github.com/timdrysdale/relay/pkg/hub"
	"github.com/timdrysdale/relay/pkg/reconws"
)

// Hub represents a messaging hub
type Hub struct {
	Messages  *agg.Hub
	Clients   map[string]*Client //map Id string to client
	Rules     map[string]Rule    //map Id string to Rule
	Add       chan Rule
	Delete    chan string      //Id string
	Broadcast chan hub.Message //for messages incoming from the websocket server(s)
}

// Rule represents a rule for outgoing messages
type Rule struct {
	ID          string `json:"id"`
	Stream      string `json:"stream"`
	Destination string `json:"destination"`
	Token       string `json:"token"`
	File        string `json:"file"`
}

// Client represents a client for sending outgoing messages
type Client struct {
	Hub       *Hub //can access messaging hub via <client>.Hub.Messages
	Messages  *hub.Client
	Context   context.Context
	Cancel    context.CancelFunc
	Websocket *reconws.ReconWs
	File      string
}
