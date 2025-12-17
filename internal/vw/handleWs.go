package vw

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/practable/relay/internal/hub"
	log "github.com/sirupsen/logrus"
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
		_ = c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)

	err := c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		log.Errorf("readPump deadline error: %v", err)
		return
	}

	c.Conn.SetPongHandler(func(string) error {
		err := c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return err
	})

	for {
		mt, data, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Tracef("unexpected error: %v", err)
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
		_ = c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Messages.Send:
			err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				log.Errorf(" writePump deadline error: %s", err.Error())
				return
			}

			if !ok {
				// The hub closed the channel.
				err := c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				if err != nil {
					log.Errorf("writePump closeMessage error: %s", err.Error())
				}
				return
			}

			w, err := c.Conn.NextWriter(message.Type)
			if err != nil {
				return
			}

			n, err := w.Write(message.Data)
			if err != nil {
				log.Errorf("writePump writing error: %v", err)
			}
			size := len(message.Data)
			if n != size {
				log.Errorf("writePump incomplete write %d of %d", n, size)
			}
			// Add queued chunks to the current websocket message, without delimiter.
			m := len(c.Messages.Send)
			for i := 0; i < m; i++ {
				followOnMessage := <-c.Messages.Send
				n, err := w.Write(followOnMessage.Data)
				if err != nil {
					log.Errorf("writePump writing error: %v", err)
				}
				size += n
				if n != size {
					log.Errorf("writePump incomplete write %d of %d", n, size)
				}
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				log.Errorf("writePump ping deadline error: %s", err.Error())
				return
			}
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Warnf("done because conn error %s", err.Error())
				return
			}
		case <-closed:
			return
		}
	}
}
