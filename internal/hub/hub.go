package hub

import (
	log "github.com/sirupsen/logrus"
)

// New returns a pointer to an initiatialised Hub
func New() *Hub {
	return &Hub{
		Broadcast:  make(chan Message),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[string]map[*Client]bool),
	}
}

// Run starts the hub
func (h *Hub) Run(closed chan struct{}) {
	for {
		select {
		case <-closed:
			return
		case client := <-h.Register:
			if _, ok := h.Clients[client.Topic]; !ok {

				log.WithField("topic", client.Topic).Debug("Creating topic in hub")

				h.Clients[client.Topic] = make(map[*Client]bool)
			}
			h.Clients[client.Topic][client] = true
		case client := <-h.Unregister:
			// no guard needed SCC-S1033
			delete(h.Clients[client.Topic], client)
			//client knows it is finished, so no need to close(client.Send)

		case message := <-h.Broadcast:
			topic := message.Sender.Topic
			for client := range h.Clients[topic] {
				if client.Name != message.Sender.Name {
					select {
					case client.Send <- message:
					default:
						//ignore log.WithField("client", client).Error("Unregistering unresponsive client")
					}
				}
			}
		}
	}
}
