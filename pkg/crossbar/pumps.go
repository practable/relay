package crossbar

import (
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {

	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
		log.Trace("readpump closed")
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {

		mt, data, err := c.conn.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("error: %v", err)
			}
			break
		}

		if c.canWrite {

			c.hub.broadcast <- message{sender: *c, data: data, mt: mt}
			c.stats.tx.mu.Lock()
			t := time.Now()
			if c.stats.tx.ns.Count() > 0 {
				c.stats.tx.ns.Add(float64(t.UnixNano() - c.stats.tx.last.UnixNano()))
			} else {
				c.stats.tx.ns.Add(float64(t.UnixNano() - c.stats.connectedAt.UnixNano()))
			}
			c.stats.tx.last = t
			c.stats.tx.size.Add(float64(len(data)))
			c.stats.tx.mu.Unlock()
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump(closed <-chan struct{}, cancelled <-chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		log.Debug("write pump dead")
	}()
	for {
		log.Debug("Write pump alive")
		select {

		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if c.canRead { //only send if authorised to read

				w, err := c.conn.NextWriter(message.mt)
				if err != nil {
					return
				}

				w.Write(message.data)

				size := len(message.data)

				// Add queued chunks to the current websocket message, without delimiter.
				// TODO check what impact, if any, this has on jsmpeg memory requirements
				// when crossbar is loaded enough to cause message queuing
				// TODO benchmark effect of loading on message queuing
				n := len(c.send)
				for i := 0; i < n; i++ {
					followOnMessage := <-c.send
					w.Write(followOnMessage.data)
					size += len(followOnMessage.data)
				}
				c.stats.rx.mu.Lock()
				t := time.Now()
				if c.stats.rx.ns.Count() > 0 {
					c.stats.rx.ns.Add(float64(t.UnixNano() - c.stats.rx.last.UnixNano()))
				} else {
					c.stats.rx.ns.Add(float64(t.UnixNano() - c.stats.connectedAt.UnixNano()))
				}
				c.stats.rx.last = t
				c.stats.rx.size.Add(float64(size))
				c.stats.rx.mu.Unlock()
				if err := w.Close(); err != nil {
					return
				}
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-closed:
			return
		case <-cancelled:
			return
		}
	}
}
