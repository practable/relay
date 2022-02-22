package agg

import (
	"github.com/practable/relay/internal/hub"
)

// Hub is the main structure representing an aggegrating hub
type Hub struct {
	Hub        *hub.Hub
	Broadcast  chan hub.Message
	Register   chan *hub.Client
	Unregister chan *hub.Client
	Add        chan Rule
	Delete     chan string
	Rules      map[string][]string
	Streams    map[string]map[*hub.Client]bool
	SubClients map[*hub.Client]map[*SubClient]bool
}

// Rule represents which Feeds a combined to form a Stream
type Rule struct {
	Stream string   `json:"stream"`
	Feeds  []string `json:"feeds"`
}

// SubClient represents subclients which are used for aggregation tasks
type SubClient struct {
	Client  *hub.Client
	Stopped chan struct{}
}
