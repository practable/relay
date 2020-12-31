package rwc

import (
	"context"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/relay/pkg/agg"
	"github.com/timdrysdale/relay/pkg/hub"
	"github.com/timdrysdale/relay/pkg/reconws"
)

// pass in the messaging hub as a parameter
// assume it is already running
func New(messages *agg.Hub) *Hub {

	h := &Hub{
		Messages: messages,
		Clients:  make(map[string]*Client), //map Id string to Client
		Rules:    make(map[string]Rule),    //map Id string to Rule
		Add:      make(chan Rule),
		Delete:   make(chan string), //Id string
	}

	return h
}

func (h *Hub) Run(closed chan struct{}) {

	defer func() {
		//on exit, go through the list of open clients and close them
		//may panic if a client is individually closed just before exiting
		//but if exiting, a panic is less of an issue
		for _, client := range h.Clients {
			client.Cancel()
		}
	}()

	for {
		select {
		case <-closed:
			return
		case rule := <-h.Add:

			if rule.Id == "deleteAll" {
				break //reserved id (for deleting all rules)
			}

			// Allow multiple destinations for a stream;
			// allow multiple streams per destination;
			// allow only one client per rule.Id.
			// Delete any pre-existing client for this rule.Id
			// because it just became superseded
			if client, ok := h.Clients[rule.Id]; ok {
				h.Messages.Unregister <- client.Messages
				client.Cancel() //stop RelayIn() & RelayOut()
				delete(h.Clients, rule.Id)
			}
			delete(h.Rules, rule.Id)

			//record the new rule for later convenience in reporting
			h.Rules[rule.Id] = rule

			// create new reconnecting websocket client
			ws := reconws.New()

			urlStr := rule.Destination //no sanity check - don't dupe ws functionality

			token := rule.Token

			// create client to handle stream messages
			messageClient := &hub.Client{Hub: h.Messages.Hub,
				Name:  rule.Destination,
				Topic: rule.Stream,
				Send:  make(chan hub.Message, 2),
				Stats: hub.NewClientStats()}

			ctx, cancel := context.WithCancel(context.Background())
			client := &Client{Hub: h,
				Messages:  messageClient,
				Context:   ctx,
				Cancel:    cancel,
				Websocket: ws,
				File:      rule.File,
			}

			h.Clients[rule.Id] = client

			h.Messages.Register <- client.Messages //register for messages from hub

			go client.RelayIn(client.Context)
			go client.RelayOut(client.Context)

			if token == "" {
				log.WithFields(log.Fields{"url": urlStr}).Trace("Starting Reconnect")
				go ws.Reconnect(client.Context, urlStr)
			} else {
				log.WithFields(log.Fields{"url": urlStr, "token": token}).Trace("Starting ReconnectAuth")
				go ws.ReconnectAuth(client.Context, urlStr, token)
			}
			//user must check stats to learn of errors
			// an RPC style return on start is of limited value because clients are long lived
			// so we'll need to check the stats later anyway; better just to do things one way

		case ruleId := <-h.Delete:

			if ruleId == "deleteAll" {
				for _, client := range h.Clients {
					h.Messages.Unregister <- client.Messages
					client.Cancel() //stop RelayIn() & RelayOut()
				}
				h.Clients = make(map[string]*Client)
				h.Rules = make(map[string]Rule)

			} else {
				if client, ok := h.Clients[ruleId]; ok {
					h.Messages.Unregister <- client.Messages
					client.Cancel() //stop RelayIn() & RelayOut()
					delete(h.Clients, ruleId)
				}

				delete(h.Rules, ruleId)

			}
		}
	}
}

//use label to break from the for?

// relay messages from the hub to the websocket client until stopped
func (c *Client) RelayOut(ctx context.Context) {

	writeToFile := false

	// avoid shadowing in if block by declaring in advance
	var f *os.File
	var err error

	if c.File != "" {
		f, err = os.Create(c.File)
		if err == nil {
			writeToFile = true
		}
		defer f.Close()
	}

LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		case msg, ok := <-c.Messages.Send:
			if ok {
				c.Websocket.Out <- reconws.WsMessage{Data: msg.Data, Type: msg.Type}
				if writeToFile {
					_, err := f.Write(msg.Data)
					if err != nil {
						writeToFile = false //assume no more writing possible
					}
				}
			}
		}
	}
}

// relay messages from websocket server to the hub until stopped
func (c *Client) RelayIn(ctx context.Context) {
LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		case msg, ok := <-c.Websocket.In:
			if ok {
				c.Hub.Messages.Broadcast <- hub.Message{Data: msg.Data, Type: msg.Type, Sender: *c.Messages, Sent: time.Now()}
			}
		}
	}
}
