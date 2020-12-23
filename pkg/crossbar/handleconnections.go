package crossbar

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

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

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// TODO
// - change auth to code-for-token
// - per ip throttling with github.com/didip/tollbooth
// - package stats.html with
// - canary connection + reporting
// - add auth to stats connection
// - tidy comments to look better in docs
// - remove stale code and comments
// - redo load testing / performance / memory leakage
// - add benchmarking

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

func fpsFromNs(ns float64) float64 {
	return 1 / (ns * 1e-9)
}

func (c *Client) statsReporter() {
	defer func() {
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.WithField("error", err).Error("statsReporter ReadMessage")
			}
			break
		}

		for _, topic := range c.hub.clients {
			for client, _ := range topic {

				//c.conn.SetWriteDeadline(time.Now().Add(writeWait))
				//
				//w, err := c.conn.NextWriter(websocket.TextMessage)
				//if err != nil {
				//	return
				//}

				var tx ReportStats

				if client.stats.tx.size.Count() > 0 {
					tx = ReportStats{
						Last: time.Since(client.stats.tx.last).String(),
						Size: math.Round(client.stats.tx.size.Mean()),
						Fps:  fpsFromNs(client.stats.tx.ns.Mean()),
					}
				} else {
					tx = ReportStats{
						Last: "Never",
						Size: 0,
						Fps:  0,
					}
				}

				var rx ReportStats

				if client.stats.rx.size.Count() > 0 {
					rx = ReportStats{
						Last: time.Since(client.stats.rx.last).String(),
						Size: math.Round(client.stats.rx.size.Mean()),
						Fps:  fpsFromNs(client.stats.rx.ns.Mean()),
					}
				} else {
					rx = ReportStats{
						Last: "Never",
						Size: 0,
						Fps:  0,
					}
				}

				report := &ClientReport{
					Topic:       client.topic,
					Broadcaster: client.broadcaster,
					Connected:   client.stats.connectedAt.String(),
					RemoteAddr:  client.remoteAddr,
					UserAgent:   client.userAgent,
					Stats: RxTx{
						Tx: tx,
						Rx: rx,
					},
				}

				b, err := json.Marshal(report)

				if err != nil {
					log.WithField("error", err).Error("statsReporter marshalling JSON")
					return
				} else {
					c.send <- message{data: b, mt: websocket.TextMessage}
					//w.Write(b)
				}

				//if err := w.Close(); err != nil {
				//	return
				//}
			}
		}

	}
}

func (c *Client) statsManager(closed <-chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(message.mt)
			if err != nil {
				return
			}

			w.Write(message.data)

			// commented out because need one object per message?
			// Add queued chunks to the current websocket message, without delimiter.
			//n := len(c.send)
			//for i := 0; i < n; i++ {
			//	followOnMessage := <-c.send
			//	w.Write(followOnMessage.data)
			//}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-closed:
			return
		}
	}
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump(broadcaster bool) {

	quit := make(chan struct{})

	// use wg to reinforce that we must kill the timer goroutine (to avoid memory leaks)
	// TODO: figure out how to test this behaviour explicitly, so it is not lost in future updates
	var wg sync.WaitGroup

	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
		log.Debug("Deferred close of readPump")
		close(quit)
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	// skip authentication if there is no secret
	if c.secret == "" {
		log.WithField("topic", c.topic).Warn("No secret supplied so authorisation is permitted for all connections")
		c.authorised = true
	}

	for {

		mt, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("error: %v", err)
			}
			break
		}

		if !c.authorised {

			// this if statement deliberately does not allow authorisation to be extended by additional auth messages sent later
			// Since we only receive one authorisation message per connection, for a spam connection it will either
			// a/ stay connected until auth attempted, receiving nothing, and using little resource
			// b/ attempt auth, fail, get disconnected, and need to reconnect - in which case we look to proxy for rate-limiting the reconnections

			// check if the message is a JWT with authentication
			// https://github.com/dgrijalva/jwt-go/issues/397: Make sure you're not clicking "secret base64 encoded" if you enter "mysecret" as the secret.

			// We convert all broadcasters connecting on /in/<route> to the topic of /out/<route>
			// so we need to adjust our topic back to /in/<route> before we can check the token
			topic := c.topic

			if c.broadcaster {
				topic = strings.Replace(topic, "/out/", "/in/", 1)
			}

			route := c.audience + topic

			now := time.Now().Unix()

			lifetime, err := checkAuth(data, c.secret, route, now)

			if err == nil {

				c.authorised = true

				wg.Add(1)

				go func() {

					defer wg.Done()

					select {
					case <-quit:
						log.WithFields(log.Fields{"topic": c.topic}).Debug("Timeout has been cancelled by quit channel")
						return
					case <-time.After(time.Duration(lifetime) * time.Second):

						c.authorised = false

						// don't send expired message to avoid jsmpeg having to check each incoming message
						// for whether it is MPEG TS or json
						// propose using separate service to send "information on token status"
						// i.e. subscribe to the token alerts service with your tokens ...
						// (which need not necessarily check the signature?)

						log.WithFields(log.Fields{"topic": c.topic}).Warn("Token has expired")
					}

				}()

				accepted := &AuthMessage{
					Topic:      topic,
					Token:      string(data),
					Authorised: true,
					Reason:     "ok",
				}

				reply, err := json.Marshal(accepted)

				if err == nil {
					c.adminSend <- message{sender: *c, data: reply, mt: websocket.TextMessage}
				} else {
					log.WithField("error", err.Error()).Warn("Could not marshal authorisation-accepted message")
				}

			} else {

				denied := &AuthMessage{
					Topic:      topic,
					Token:      string(data),
					Authorised: false,
					Reason:     err.Error(), //"denied",
				}

				reply, err := json.Marshal(denied)

				if err == nil {
					log.WithField("topic", topic).Debug("Sending auth denied message")
					c.adminSend <- message{sender: *c, data: reply, mt: websocket.TextMessage}
					log.WithField("topic", topic).Info("Sent auth denied message")
				} else {
					log.WithField("error", err.Error()).Warn("Could not marshal authorisation-denied message")
				}
				time.Sleep(50 * time.Millisecond) // allow some time for denied message to be sent
				log.WithField("topic", c.topic).Warn("Auth incorrect, closing connection")
				return // close connection if auth is incorrect. Will this mess up

			}
		} else if broadcaster { // do in this order to save broadcasting the auth message! (TODO - add test for this)

			c.hub.broadcast <- message{sender: *c, data: data, mt: mt}

			t := time.Now()
			if c.stats.tx.ns.Count() > 0 {
				c.stats.tx.ns.Add(float64(t.UnixNano() - c.stats.tx.last.UnixNano()))
			} else {
				c.stats.tx.ns.Add(float64(t.UnixNano() - c.stats.connectedAt.UnixNano()))
			}
			c.stats.tx.last = t
			c.stats.tx.size.Add(float64(len(data)))

		}
	}

	wg.Wait()
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump(closed <-chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		log.Debug("write pump dead")
	}()
	for {
		log.Debug("Write pump alive")
		select {

		case message, ok := <-c.adminSend:
			log.Debug("Sending admin message")
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(message.mt)
			if err != nil {
				return
			}

			w.Write(message.data)
			if err := w.Close(); err != nil {
				return
			}
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if c.authorised { //only send if authorised

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
				return
			}
		case <-closed:
			return
		}
	}
}

func serveStats(closed <-chan struct{}, hub *Hub, w http.ResponseWriter, r *http.Request, config Config) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithField("error", err).Error("Upgrading serveStats")
		return
	}

	// TODO check authorisation to get stats
	log.Warn("Stats channel not protected; is open to all")

	client := &Client{hub: hub,
		conn: conn,
		send: make(chan message, 256),
	}
	go client.statsReporter()
	go client.statsManager(closed)
}

func servePage(w http.ResponseWriter, r *http.Request) {
	// TODO use gobuffalo/packr to embed this file or wait for go1.16
	log.Println(r.URL)
	if r.URL.Path != "/stats" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "stats.html")
}

func HandleConnections(closed <-chan struct{}, wg *sync.WaitGroup, clientActionsChan chan clientAction, messagesFromMe chan message, config Config) {
	hub := newHub()
	go hub.run()

	http.HandleFunc("/stats", servePage)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveWs(closed, hub, w, r, config)
	})
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveStats(closed, hub, w, r, config)
	})

	h := &http.Server{Addr: config.Addr, Handler: nil}

	go func() {
		if err := h.ListenAndServe(); err != nil {
			log.Info("ListenAndServe: ", err) //TODO upgrade to fatal once httptest is supported
		}
	}()

	<-closed

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h.Shutdown(ctx)
	wg.Done()
}
