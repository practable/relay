package agg

import (
	"strings"
	"sync"

	"github.com/jinzhu/copier"
	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/relay/pkg/hub"
)

func New() *Hub {

	h := &Hub{
		Hub:        hub.New(),
		Broadcast:  make(chan hub.Message),
		Register:   make(chan *hub.Client),
		Unregister: make(chan *hub.Client),
		Streams:    make(map[string]map[*hub.Client]bool),
		SubClients: make(map[*hub.Client]map[*SubClient]bool),
		Rules:      make(map[string][]string),
		Add:        make(chan Rule),
		Delete:     make(chan string),
	}

	return h

}

func (h *Hub) Run(closed chan struct{}) {
	h.RunOptionalStats(closed, false)
}

func (h *Hub) RunWithStats(closed chan struct{}) {
	h.RunOptionalStats(closed, true)
}

func (h *Hub) RunOptionalStats(closed chan struct{}, withStats bool) {

	//start the hub
	if withStats {
		go h.Hub.RunWithStats(closed)
	} else {
		go h.Hub.Run(closed)
	}

	for {
		select {
		case <-closed:
			return
		case client := <-h.Register:
			if strings.HasPrefix(client.Topic, "stream/") {
				// register the client to the stream
				if _, ok := h.Streams[client.Topic]; !ok {
					h.Streams[client.Topic] = make(map[*hub.Client]bool)
				}
				h.Streams[client.Topic][client] = true

				// register the client to any feeds currently set by stream rule
				if feeds, ok := h.Rules[client.Topic]; ok {
					h.SubClients[client] = make(map[*SubClient]bool)
					wg := &sync.WaitGroup{}
					for _, feed := range feeds {
						// create and store the subclients we will register with the hub
						subClient := &SubClient{Client: &hub.Client{}}

						err := copier.Copy(&subClient.Client, client)
						if err != nil {
							log.Errorf("subclient copy error %s", err.Error())
						}
						subClient.Client.Topic = feed
						subClient.Client.Send = make(chan hub.Message)
						subClient.Stopped = make(chan struct{})
						h.SubClients[client][subClient] = true
						wg.Add(1)
						go subClient.RelayTo(client)
						h.Hub.Register <- subClient.Client
					}

				}
			} else {
				// register client directly
				h.Hub.Register <- client
			}
		case client := <-h.Unregister:
			if strings.HasPrefix(client.Topic, "stream/") {
				// unregister any subclients that are registered to feeds
				for subClient := range h.SubClients[client] {
					h.Hub.Unregister <- subClient.Client
					close(subClient.Stopped)
				}

				// delete the client from the stream
				if _, ok := h.Streams[client.Topic]; ok {
					delete(h.Streams[client.Topic], client)
					//close(client.Send)
				}
				delete(h.SubClients, client)

			} else {
				// unregister client directly
				h.Hub.Unregister <- client
			}
		case msg := <-h.Broadcast:
			// defer handling to hub
			// note that non-responsive clients will get deleted
			h.Hub.Broadcast <- msg
		case rule := <-h.Add:
			if rule.Stream == "deleteAll" {
				break //reserved ID for deleting all rules
			}
			// unregister clients from old feeds, if any
			if _, ok := h.Rules[rule.Stream]; ok {
				for client := range h.Streams[rule.Stream] {
					for subClient := range h.SubClients[client] {
						h.Hub.Unregister <- subClient.Client
						close(subClient.Stopped)
					}
				}
			}
			//set new rule
			h.Rules[rule.Stream] = rule.Feeds
			// register the clients to any feeds currently set by stream rule
			if feeds, ok := h.Rules[rule.Stream]; ok {
				for client := range h.Streams[rule.Stream] {
					h.SubClients[client] = make(map[*SubClient]bool)
					for _, feed := range feeds {
						// create and store the subclients we will register with the hub
						subClient := &SubClient{Client: &hub.Client{}}
						err := copier.Copy(&subClient.Client, client)
						if err != nil {
							log.Errorf("subclient copy error %s", err.Error())
						}
						subClient.Client.Topic = feed
						subClient.Client.Send = make(chan hub.Message)
						subClient.Stopped = make(chan struct{})
						h.SubClients[client][subClient] = true
						go subClient.RelayTo(client)
						h.Hub.Register <- subClient.Client
					}
				}
			}

		case stream := <-h.Delete:

			if stream == "deleteAll" { //all streams to be deleted

				for _, client := range h.SubClients {
					for subclient := range client { //h.SubClients[client] {
						h.Hub.Unregister <- subclient.Client
						close(subclient.Stopped)
					}
				}

				h.Rules = make(map[string][]string)

			} else { //single stream

				// unregister clients from old feeds, if any
				if _, ok := h.Rules[stream]; ok {
					for client := range h.Streams[stream] {
						for subClient := range h.SubClients[client] {
							h.Hub.Unregister <- subClient.Client
							close(subClient.Stopped)
						}
					}
				}

				// delete rule
				delete(h.Rules, stream)
			}
		}
	}
}

// relay messages from subClient to Client
func (sc *SubClient) RelayTo(c *hub.Client) {
	for {
		select {
		case <-sc.Stopped:
			return
		case msg, ok := <-sc.Client.Send:
			if ok {
				c.Send <- msg
			} else {
				return
			}
		}
	}
}
