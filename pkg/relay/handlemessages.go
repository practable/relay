package cmd

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[string]map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan message

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[string]map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			if _, ok := h.clients[client.topic]; !ok {
				h.clients[client.topic] = make(map[*Client]bool)
			}
			h.clients[client.topic][client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client.topic]; ok {
				delete(h.clients[client.topic], client)
				close(client.send)
			}
		case message := <-h.broadcast:
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
		}
	}
}
