package agg

import (
	"github.com/timdrysdale/relay/pkg/hub"
)

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

type Rule struct {
	Stream string   `json:"stream"`
	Feeds  []string `json:"feeds"`
}

type SubClient struct {
	Client  *hub.Client
	Stopped chan struct{}
}
