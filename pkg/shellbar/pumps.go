package shellbar

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {

	id := "shellbar.readPump(" + c.topic + "/" + c.name + ")"

	defer func() {
		log.Tracef("%s.defer(): about to disconnect", id)
		// Tell the host that we have gone ...

		// alert SSH host agent to make a new connection to relay at the same address
		// No stats needed because we are not registering to receive messages
		adminClient := &Client{
			topic: getHostTopicFromUniqueTopic(c.topic),
			name:  uuid.New().String(),
		}

		ca := ConnectionAction{
			Action: "disconnect",
			UUID:   c.hostAlertUUID,
		}

		camsg, err := json.Marshal(ca)

		if err != nil {
			log.WithFields(log.Fields{"error": err, "uuid": c.hostAlertUUID}).Errorf("%s.defer(): Failed to make disconnect connectionAction message because %s", id, err.Error())
			return
		}

		c.hub.broadcast <- message{sender: *adminClient, data: camsg, mt: websocket.TextMessage}
		log.Tracef("%s.defer(): broadcast disconnect of UUID %s", id, c.hostAlertUUID)

		c.hub.unregister <- c
		log.Tracef("%s.defer(): client unregistered", id)

		c.conn.Close()
		log.Tracef("%s.defer(): DONE", id)

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

		size := len(data)

		if c.canWrite {

			c.hub.broadcast <- message{sender: *c, data: data, mt: mt}

			log.WithFields(log.Fields{"topic": c.topic, "size": size}).Tracef("%s: broadacast %d-byte message to topic %s", id, size, c.topic)

			t := time.Now()
			if c.stats.tx.ns.Count() > 0 {
				c.stats.tx.ns.Add(float64(t.UnixNano() - c.stats.tx.last.UnixNano()))
			} else {
				c.stats.tx.ns.Add(float64(t.UnixNano() - c.stats.connectedAt.UnixNano()))
			}
			c.stats.tx.last = t
			c.stats.tx.size.Add(float64(len(data)))

		} else {
			log.WithFields(log.Fields{"topic": c.topic, "size": size}).Tracef("%s: ignored %d-byte message intended for broadcast to topic %s", id, size, c.topic)

		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump(closed <-chan struct{}, cancelled <-chan struct{}) {

	id := "shellbar.writePump(" + c.topic + "/" + c.name + ")"

	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		log.Tracef("%s: done", id)
	}()
	log.Tracef("%s: starting", id)
	for {

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

				log.WithFields(log.Fields{"topic": c.topic, "size": size}).Tracef("%s: wrote %d-byte message from topic %s", id, size, c.topic)

				// don't queue chunks; makes reading JSON objects on the host connectAction channel fail if two connects happen together

				t := time.Now()
				if c.stats.rx.ns.Count() > 0 {
					c.stats.rx.ns.Add(float64(t.UnixNano() - c.stats.rx.last.UnixNano()))
				} else {
					c.stats.rx.ns.Add(float64(t.UnixNano() - c.stats.connectedAt.UnixNano()))
				}
				c.stats.rx.last = t
				c.stats.rx.size.Add(float64(size))

				if err := w.Close(); err != nil {
					return
				}
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Warnf("%s: done because conn error %s", id, err.Error())
				return
			}
		case <-closed:
			log.Tracef("%s: done because closed channel closed", id)
			return
		case <-cancelled:
			log.Tracef("%s: done because cancelled channel closed", id)
			return
		}
	}
}
