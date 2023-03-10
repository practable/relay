/*
   client is a public wrapper for internal/reconws so that
   we can expose this useful code but without having to
   commit to publically declaring the specifics of the
   internal API, as this may change later.
*/

package client

import (
	"context"

	"github.com/gorilla/websocket"
	"github.com/practable/relay/internal/reconws"
)

const (
	TextMessage   int = websocket.TextMessage
	BinaryMessage     = websocket.BinaryMessage
)

type Message struct {
	Content []byte
	Type    int // TextMessage or BinaryMessage
}

type Client struct {
	r       *reconws.ReconWs
	Receive chan Message
	Send    chan Message
}

func New() *Client {
	return &Client{
		r:       reconws.New(),
		Receive: make(chan Message),
		Send:    make(chan Message),
	}
}

func (c *Client) Connect(ctx context.Context, to, token string) {
	go func() {
	LOOP:
		for {
			select {
			case <-ctx.Done():
				break LOOP //we're done
			case msg := <-c.r.In:
				c.Receive <- Message{Content: msg.Data, Type: msg.Type}
			}
		}
	}()
	go func() {
	LOOP:
		for {
			select {
			case <-ctx.Done():
				break LOOP //we're done
			case msg := <-c.Send:
				c.r.Out <- reconws.WsMessage{Data: msg.Content, Type: msg.Type}
			}
		}
	}()
	c.r.ReconnectAuth(ctx, to, token)
}
