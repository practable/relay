package cmd

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/relay/pkg/hub"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer (10MB)
	// Typical key frame at 640x480 is 60 * 188B ~= 11kB
	maxMessageSize = 1024 * 1024 * 10
)

// 4096 Bytes is the approx average message size
// this number does not limit message size
// So for key frames we just make a few more syscalls
// null subprotocol required by Chrome
// TODO restrict CheckOrigin
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	Subprotocols:    []string{"null"},
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func (app *App) handleWs(w http.ResponseWriter, r *http.Request) {

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithField("error", err).Error("Failed upgrading to websocket connection in wsHandler")
		return
	}

	vars := mux.Vars(r)
	topic := vars["feed"]

	messageClient := &hub.Client{Hub: app.Hub.Hub,
		Name:  uuid.New().String()[:3],
		Send:  make(chan hub.Message),
		Stats: hub.NewClientStats(),
		Topic: topic,
	}

	client := &WsHandlerClient{
		Messages:   messageClient,
		Conn:       conn,
		UserAgent:  r.UserAgent(),
		RemoteAddr: r.Header.Get("X-Forwarded-For"),
	}

	app.Hub.Register <- client.Messages

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump(app.Closed)
	go client.readPump()

}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *WsHandlerClient) readPump() {
	defer func() {
		c.Messages.Hub.Unregister <- c.Messages
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		mt, data, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("error: %v", err)
			}
			break
		}

		t := time.Now()

		c.Messages.Hub.Broadcast <- hub.Message{Sender: *c.Messages, Data: data, Type: mt, Sent: t}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *WsHandlerClient) writePump(closed <-chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Messages.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(message.Type)
			if err != nil {
				return
			}

			w.Write(message.Data)

			size := len(message.Data)

			// Add queued chunks to the current websocket message, without delimiter.
			n := len(c.Messages.Send)
			for i := 0; i < n; i++ {
				followOnMessage := <-c.Messages.Send
				w.Write(followOnMessage.Data)
				size += len(followOnMessage.Data)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-closed:
			return
		}
	}
}
