package crossbar

import (
	"sync"

	"github.com/practable/relay/internal/chanmap"
	log "github.com/sirupsen/logrus"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[string]map[*Client]bool

	// deny channel store
	dcs *chanmap.Store

	mu *sync.RWMutex

	// Inbound messages from the clients.
	broadcast chan message

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		mu:         &sync.RWMutex{},
		broadcast:  make(chan message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[string]map[*Client]bool),
	}
}

// SetDenyChannelStore adds a pointer to the channel map store to the hub
func (h *Hub) SetDenyChannelStore(dcs *chanmap.Store) {
	h.dcs = dcs
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if _, ok := h.clients[client.topic]; !ok {
				h.clients[client.topic] = make(map[*Client]bool)
			}
			h.clients[client.topic][client] = true
			h.mu.Unlock()
			err := h.dcs.Add(client.bookingID, client.name, client.denied)
			if err != nil {
				log.WithFields(log.Fields{"error": err.Error(), "topic": client.topic, "booking_id": client.bookingID}).Warning("deny channel not added on client register")
			}
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.topic]; ok {
				delete(h.clients[client.topic], client)
				close(client.send)
			}
			h.mu.Unlock()
			err := h.dcs.DeleteChild(client.name) // no need to close, not denied
			if err != nil {
				log.WithFields(log.Fields{"error": err.Error(), "topic": client.topic, "booking_id": client.bookingID}).Warning("deny channel not deleted on client unregister")
			}
		case message := <-h.broadcast:
			h.mu.RLock()
			topic := message.sender.topic
			for client := range h.clients[topic] {
				if client.name != message.sender.name {
					select {
					case client.send <- message:
					default:
						h.unregister <- client
						//close(client.send)
						//delete(h.clients[topic], client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}
