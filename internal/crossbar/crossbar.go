// Package crossbar provides a message broker with topics
package crossbar

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/practable/relay/internal/chanmap"
	"github.com/practable/relay/internal/deny"
	"github.com/practable/relay/internal/permission"
	"github.com/practable/relay/internal/ttlcode"
	"github.com/practable/relay/internal/util"
	log "github.com/sirupsen/logrus"
)

// Config represents configuration options for a crossbar instance
// Use this struct to pass configuration as argument during testing
type Config struct {

	// Audience must match the host in token
	Audience string

	//BufferSize sets the buffer size for client communications channels
	BufferSize int64

	// ExchangeCode swaps a code for the associated Token
	CodeStore *ttlcode.CodeStore

	//DenyStore holds deny-listed bookingIDs
	DenyStore *deny.Store

	//Hub holds the clients and topics and manages message distribution
	Hub *Hub

	// Listen is the listening port
	Listen int

	// Secret is used to validating statsTokens
	Secret string

	//StatsEvery sets how often stats are reported
	StatsEvery time.Duration
}

// Client is a middleperson between the websocket connection and the hub.
type Client struct {

	// bookingID from the token
	bookingID string

	// the time we accepted the connection from the client
	connectedAt int64

	// hub closes this channel if connection is curtailed
	denied chan struct{}

	// when the client's authorization token expires
	expiresAt int64

	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan message

	// string representing the path the client connected to
	topic string

	audience string

	scopes []string

	name string

	userAgent string

	remoteAddr string

	// existence of scopes to read, write
	canRead, canWrite bool
}

// ClientReport represents information about a client's connection, permissions, and statistics
type ClientReport struct {
	CanRead bool `json:"canRead"`

	CanWrite bool `json:"canWrite"`

	ConnectedAt string `json:"connected"`

	ExpiresAt string `json:"expiresAt"`

	RemoteAddr string `json:"remoteAddr"`

	Scopes []string `json:"scopes"`

	Topic string `json:"topic"`

	UserAgent string `json:"userAgent"`
}

// StatsCommand represents a command in string form
type StatsCommand struct {
	Command string `json:"cmd"`
}

// messages will be wrapped in this struct for muxing
type message struct {
	sender Client
	mt     int
	data   []byte //text data are converted to/from bytes as needed
}

// NewDefaultConfig returns a pointer to a Config struct with default parameters
func NewDefaultConfig() *Config {
	c := &Config{}
	c.Listen = 3000
	c.CodeStore = ttlcode.NewDefaultCodeStore()
	c.BufferSize = 128
	c.StatsEvery = time.Duration(5 * time.Second)
	log.WithFields(log.Fields{"BufferSize": c.BufferSize, "listen": c.Listen, "ttl": c.CodeStore.GetTTL()}).Info("crossbar default config")
	return c
}

// WithBufferSize specifies the client channel buffer size
func (c *Config) WithBufferSize(n int64) *Config {
	if n > 0 && n <= 512 {
		c.BufferSize = n
	} else {
		log.WithFields(log.Fields{"requested": n, "actual": c.BufferSize}).Error("BufferSize must be between 1 - 512 (128 recommended)")
	}
	log.WithFields(log.Fields{"BufferSize": n}).Info("crossbar buffer size set")
	return c
}

// WithStatsEvery specifies how often to send stats
func (c *Config) WithStatsEvery(statsEvery time.Duration) *Config {
	c.StatsEvery = statsEvery
	log.WithFields(log.Fields{"statsEvery": statsEvery}).Info("crossbar statsEvery set")
	return c
}

// WithListen specified which (int) port to listen on
func (c *Config) WithListen(listen int) *Config {
	c.Listen = listen
	log.WithFields(log.Fields{"listen": listen}).Info("crossbar listening port set")
	return c
}

// WithAudience specificies the audience for the tokens
func (c *Config) WithAudience(audience string) *Config {
	c.Audience = audience
	log.WithFields(log.Fields{"audience": audience}).Info("crossbar audience set")
	return c
}

// WithCodeStoreTTL specifies the lifetime for the codestore
func (c *Config) WithCodeStoreTTL(ttl int64) *Config {
	c.CodeStore = ttlcode.NewDefaultCodeStore().
		WithTTL(ttl)
	log.WithFields(log.Fields{"ttl": ttl}).Info("crossbar ttl set")
	return c
}

// Crossbar creates and runs a new crossbar instance
func Crossbar(config Config, closed <-chan struct{}, denied chan string, parentwg *sync.WaitGroup) {

	var wg sync.WaitGroup

	messagesToDistribute := make(chan message, 10) //TODO make buffer length configurable

	wg.Add(1)

	go handleConnections(closed, &wg, messagesToDistribute, denied, config)

	wg.Wait()

	parentwg.Done()

}

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

	err := c.conn.SetReadDeadline(time.Now().Add(pongWait))

	if err != nil {
		log.Errorf("readPump deadline error: %v", err)
		return
	}

	c.conn.SetPongHandler(func(string) error {
		err := c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return err
	})

	for {

		mt, data, err := c.conn.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Tracef("readPump error: %v", err)
			}
			break
		}

		if c.canWrite {

			c.hub.broadcast <- message{sender: *c, data: data, mt: mt}

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
		log.Trace("write pump dead")
	}()
	for {
		log.Trace("write pump alive")
		select {

		case message, ok := <-c.send:
			err := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				log.Errorf("writePump deadline error: %s", err.Error())
				return
			}

			if !ok {
				// The hub closed the channel.
				err := c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				if err != nil {
					// this error not important as channel is closed or closing anyway
					log.Tracef("writePump closeMessage error: %s", err.Error())
				}
				return
			}

			if c.canRead { //only send if authorised to read

				w, err := c.conn.NextWriter(message.mt)
				if err != nil {
					return
				}

				n, err := w.Write(message.data)

				if err != nil {
					log.Tracef("writePump writing error: %v", err)
				}

				size := len(message.data)

				if err == nil && n != size {
					log.Errorf("writePump incomplete write %d of %d", n, size) //don't log this if already a writing error
				}

				// Add queued chunks to the current websocket message, without delimiter.
				// TODO check what impact, if any, this has on jsmpeg memory requirements
				// when crossbar is loaded enough to cause message queuing
				// TODO benchmark effect of loading on message queuing
				m := len(c.send)
				for i := 0; i < m; i++ {
					followOnMessage := <-c.send

					n, err := w.Write(followOnMessage.data)
					if err != nil {
						log.WithField("error", err.Error()).Error("writePump writing error for follow on message")
					}

					if err == nil && n != len(followOnMessage.data) {
						log.WithFields(log.Fields{"wanted": size, "actual": n}).Error("writePump incomplete write")
					}

					size += n
				}

				log.Tracef("writePump wrote %d bytes", size)

				if err := w.Close(); err != nil {
					return
				}
			}
		case <-ticker.C:
			err := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				log.Errorf("writePump ping deadline error: %v", err)
				return
			}
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

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[string]map[*Client]bool

	// deny channel store
	dcs *chanmap.Store

	mu *sync.RWMutex

	// Inbound messages from the clients.
	broadcast chan message

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func New() *Hub {
	return newHub()
}

func newHub() *Hub {
	return &Hub{
		mu:         &sync.RWMutex{},
		broadcast:  make(chan message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[string]map[*Client]bool),
	}
}

func (h *Hub) GetClientReports() []*ClientReport {

	var reports []*ClientReport

	h.mu.RLock()
	for _, topic := range h.clients {
		for client := range topic {

			ca, err := time.Unix(client.connectedAt, 0).UTC().MarshalText()
			if err != nil {
				log.WithFields(log.Fields{"error": err.Error(), "topic": client.topic, "connectedAt": client.connectedAt}).Error("stats cannot marshal connectedAt time to string")
			}
			ea, err := time.Unix(client.expiresAt, 0).UTC().MarshalText()
			if err != nil {
				log.WithFields(log.Fields{"error": err.Error(), "topic": client.topic, "expiresAt": client.expiresAt}).Error("stats cannot marshal expiresAt time to string")
			}

			report := &ClientReport{
				Topic:       client.topic,
				CanRead:     client.canRead,
				CanWrite:    client.canWrite,
				ConnectedAt: string(ca),
				ExpiresAt:   string(ea),
				RemoteAddr:  client.remoteAddr,
				Scopes:      client.scopes,
				UserAgent:   client.userAgent,
			}

			reports = append(reports, report)

		} //for client in topic
	} // for topic in hub
	h.mu.RUnlock()
	return reports

}

// SetDenyChannelStore adds a pointer to the channel map store to the hub
func (h *Hub) SetDenyChannelStore(dcs *chanmap.Store) {
	h.dcs = dcs
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if _, ok := h.clients[client.topic]; !ok {
				h.clients[client.topic] = make(map[*Client]bool)
			}
			h.clients[client.topic][client] = true
			h.mu.Unlock()
			err := h.dcs.Add(client.bookingID, client.name, client.denied)
			if err != nil {
				log.WithFields(log.Fields{"error": err.Error(), "topic": client.topic, "booking_id": client.bookingID}).Warning("deny channel not added on client register")
			}
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.topic]; ok {
				delete(h.clients[client.topic], client)
				close(client.send)
			}
			h.mu.Unlock()
			err := h.dcs.DeleteChild(client.name) // no need to close, not denied
			if err != nil {
				log.WithFields(log.Fields{"error": err.Error(), "topic": client.topic, "booking_id": client.bookingID}).Warning("deny channel not deleted on client unregister")
			}
		case message := <-h.broadcast:
			h.mu.RLock()
			topic := message.sender.topic
			for client := range h.clients[topic] {
				if client.name != message.sender.name {
					select {
					case client.send <- message:
					default:
						log.WithFields(log.Fields{
							"topic":          client.topic,
							"name":           client.name,
							"remote address": client.remoteAddr,
							"user agent":     client.userAgent,
						}).Error("message not sent because client.send was blocked")
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// ConnectionType represents whether the connection is session, shell, or unsupported
type ConnectionType int

// Session, Shell and Unsupported are enumerated values of ConnectionType
const (
	Session ConnectionType = iota
	Shell
	Unsupported
)

// serveWs handles websocket requests from clients.
func serveWs(closed <-chan struct{}, w http.ResponseWriter, r *http.Request, config Config) {

	// check if topic is of a supported type before we go any further
	ct := Unsupported

	path := slashify(r.URL.Path)

	prefix := getConnectionTypeFromPath(path)
	topic := getTopicFromPath(path)

	if prefix == "session" {
		ct = Session
	}
	if prefix == "shell" {
		ct = Unsupported //TODO implement shell!
	}

	if ct == Unsupported {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		log.WithFields(log.Fields{"connection_type": prefix, "path": path}).Error("new connection rejected because connectionType unsupported")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithFields(log.Fields{"path": path, "error": err.Error()}).Error("new connection failed to upgrade to websocket")
		return
	}

	log.WithFields(log.Fields{"topic": topic}).Debug("new connection upgraded to websocket") //Cannot return any http responses from here on

	// Enforce permissions by exchanging the authcode for a connection ticket
	// which contains expiry time, route, and permissions

	// Get the first code query param, lowercase only

	code := r.URL.Query().Get("code")

	// if no code or empty, return 401
	if code == "" {
		log.WithFields(log.Fields{"topic": topic}).Error("unauthorized because no code")
		return
	}

	// Exchange code for token

	token, err := config.CodeStore.ExchangeCode(code)

	if err != nil {
		log.WithFields(log.Fields{"error": err.Error(), "topic": topic, "booking_id": token.BookingID}).Error("unauthorized because invalid code")
		return
	}

	// if debugging, we want to show the token
	log.WithFields(log.Fields{"topic": topic, "token": util.Compact(token)}).Debug("code exchanged ok")

	// check token is a permission token so we can process it properly
	// It's been validated so we don't need to re-do that
	if !permission.HasRequiredClaims(token) {
		log.WithFields(log.Fields{"topic": topic, "booking_id": token.BookingID}).Error("unauthorized because token missing claims")
		return
	}

	now := config.CodeStore.GetTime()

	if token.NotBefore.After(time.Unix(now, 0)) {
		log.WithFields(log.Fields{"topic": topic, "booking_id": token.BookingID}).Error("unauthorized because too early")
		return
	}

	ttl := token.ExpiresAt.Unix() - now

	audok := false

	for _, aud := range token.Audience {
		if aud == config.Audience {
			audok = true
		}
	}

	topicBad := (topic != token.Topic)
	expired := ttl < 0

	if (!audok) || topicBad || expired {
		log.WithFields(log.Fields{"audience_ok": audok, "topic_ok": !topicBad, "expired": expired, "topic": topic, "booking_id": token.BookingID}).Error("unauthorized because token invalid")
		return
	}

	// we must check the booking is not denied here, else a user could request access, get a code, cancel booking, then use code to start a connection
	if config.DenyStore.IsDenied(token.BookingID) {
		log.WithFields(log.Fields{"topic": topic, "booking_id": token.BookingID}).Error("unauthorized because booking_id is deny listed")
		return
	}

	// check permissions

	var canRead, canWrite bool

	for _, scope := range token.Scopes {
		if scope == "read" {
			canRead = true
		}
		if scope == "write" {
			canWrite = true
		}
	}

	if !(canRead || canWrite) {
		log.WithFields(log.Fields{"topic": topic, "booking_id": token.BookingID, "scopes": token.Scopes}).Error("unauthorized because no valid scopes in token")
		return
	}

	cancelled := make(chan struct{})
	denied := make(chan struct{})

	if ct == Session {

		// Create a client
		client := &Client{hub: config.Hub,
			bookingID:   token.BookingID,
			conn:        conn,
			denied:      denied,
			connectedAt: time.Now().Unix(),
			expiresAt:   (*token.ExpiresAt).Unix(), // jwt.NumericDate underlying type is time.Time
			send:        make(chan message, int(config.BufferSize)),
			topic:       topic,
			name:        uuid.New().String(),
			userAgent:   r.UserAgent(),
			remoteAddr:  r.Header.Get("X-Forwarded-For"),
			audience:    config.Audience,
			canRead:     canRead,
			canWrite:    canWrite,
			scopes:      token.Scopes,
		}
		client.hub.register <- client

		cf := log.Fields{
			"booking_id":   token.BookingID,
			"connected_at": time.Unix(client.connectedAt, 0).String(),
			"expires_at":   time.Unix(client.expiresAt, 0).String(),
			"topic":        topic,
			"stats":        true,
			"buffer_size":  config.BufferSize,
			"name":         uuid.New().String(),
			"user_agent":   r.UserAgent(),
			"remote_addr":  r.Header.Get("X-Forwarded-For"),
			"audience":     config.Audience,
			"can_read":     canRead,
			"can_write":    canWrite,
			"scopes":       token.Scopes,
		}

		log.WithFields(cf).Infof("new connection")

		// cancel the connection when the token has expired or when session is curtailed
		go func() {

			select {
			case <-time.After(time.Duration(ttl) * time.Second):
				log.WithFields(cf).WithField("reason", "token expired").Info("connection closed")
			case <-denied:
				log.WithFields(cf).WithField("reason", "token denied").Info("connection closed")
			}

			close(cancelled)
		}()

		go client.writePump(closed, cancelled)
		go client.readPump()
		return
	}

}

// StatsClient starts a routine which sends stats reports on demand.
func statsClient(closed <-chan struct{}, wg *sync.WaitGroup, config Config) {

	client := &Client{hub: config.Hub,
		connectedAt: time.Now().Unix(),
		send:        make(chan message, 256),
		topic:       "stats",
		name:        "stats-generator-" + uuid.New().String(),
		audience:    config.Audience,
		userAgent:   "crossbar",
		remoteAddr:  "internal",
		canRead:     true,
		canWrite:    true,
		scopes:      []string{"read", "stats", "write"},
	}
	client.hub.register <- client

	go client.statsReporter(closed, wg, config.StatsEvery)

}

// StatsReporter sends a stats update in response to {"cmd":"update"}.
func (c *Client) statsReporter(closed <-chan struct{}, wg *sync.WaitGroup, statsEvery time.Duration) {

	defer wg.Done()

	var sc StatsCommand

	for {

		// TODO consider this scheme again
		// update on receiving a message
		// or if 60 seconds since last update
		// whichever is sooner
		// newly connecting pages can send a message
		time.Sleep(time.Second) //rate limit to one message a second
		select {
		case <-closed:
			log.Trace("StatsReporter closed")
			return
		case msg, ok := <-c.send: // received a message from hub

			if !ok {
				return //send is closed, so we are finished
			}

			err := json.Unmarshal(msg.data, &sc)

			if err != nil {
				log.WithFields(log.Fields{"error": err, "msg": string(msg.data)}).Trace("statsReporter could not marshall into json")
			}

			log.WithField("cmd", sc.Command).Trace("statsReporter received command")

			doUpdate := false

			if sc.Command == "update" {
				doUpdate = true
			}

			n := len(c.send)
			for i := 0; i < n; i++ {
				msg, ok = <-c.send
				if !ok {
					return //send is closed, so we are finished
				}

				err = json.Unmarshal(msg.data, &sc)

				if err != nil {
					log.WithFields(log.Fields{"error": err, "msg": string(msg.data)}).Trace("statsReporter could not marshall into json")
				}

				log.WithField("cmd", sc.Command).Trace("statsReporter received command")

				if sc.Command == "update" {
					doUpdate = true
				}
			}

			log.WithField("doUpdate", doUpdate).Trace("statsReporter do update?")

			if !doUpdate { //don't send updated stats
				continue
			}

		case <-time.After(statsEvery):
			log.Trace("StatsReporter routine send...")
		}

		var reports []*ClientReport
		c.hub.mu.RLock()
		for _, topic := range c.hub.clients {
			for client := range topic {

				ca, err := time.Unix(client.connectedAt, 0).UTC().MarshalText()
				if err != nil {
					log.WithFields(log.Fields{"error": err.Error(), "topic": client.topic, "connectedAt": client.connectedAt}).Error("stats cannot marshal connectedAt time to string")
				}
				ea, err := time.Unix(client.expiresAt, 0).UTC().MarshalText()
				if err != nil {
					log.WithFields(log.Fields{"error": err.Error(), "topic": client.topic, "expiresAt": client.expiresAt}).Error("stats cannot marshal expiresAt time to string")
				}

				report := &ClientReport{
					Topic:       client.topic,
					CanRead:     client.canRead,
					CanWrite:    client.canWrite,
					ConnectedAt: string(ca),
					ExpiresAt:   string(ea),
					RemoteAddr:  client.remoteAddr,
					Scopes:      client.scopes,
					UserAgent:   client.userAgent,
				}

				reports = append(reports, report)

			} //for client in topic
		} // for topic in hub
		c.hub.mu.RUnlock()
		reportsData, err := json.Marshal(reports)
		if err != nil {
			log.WithField("error", err).Error("statsReporter marshalling JSON")
			return
		}
		// broadcast stats back to the hub (i.e. and anyone listening to this topic)
		c.hub.broadcast <- message{sender: *c, data: reportsData, mt: websocket.TextMessage}

	}
}

func slashify(path string) string {

	//remove trailing slash (that's for directories)
	path = strings.TrimSuffix(path, "/")

	//ensure leading slash without needing it in config
	path = strings.TrimPrefix(path, "/")
	path = fmt.Sprintf("/%s", path)

	return path

}

func getConnectionTypeFromPath(path string) string {

	re := regexp.MustCompile(`^\/([\w\%-]*)`)

	matches := re.FindStringSubmatch(path)

	if len(matches) < 2 {
		return ""
	}

	// matches[0] = "/{prefix}/"
	// matches[1] = "{prefix}"
	return matches[1]
}

func getTopicFromPath(path string) string {

	re := regexp.MustCompile(`^\/[\w\%-]*\/([\w\%-\/]*)`)
	matches := re.FindStringSubmatch(path)

	if len(matches) < 2 {
		return ""
	}

	return matches[1]
}

func getSessionIDFromPath(path string) string {

	re := regexp.MustCompile(`^\/[\w\%-]*\/([\w\%-]*)`)
	matches := re.FindStringSubmatch(path)

	if len(matches) < 2 {
		return ""
	}

	return matches[1]
}

func getConnectionIDFromPath(path string) string {

	re := regexp.MustCompile(`^\/(?:([\w\%-]*)\/){2}([\w\%-]*)`)
	matches := re.FindStringSubmatch(path)

	if len(matches) < 2 {
		return ""
	}

	return matches[2]
}

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

// TODO
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

func handleConnections(closed <-chan struct{}, parentwg *sync.WaitGroup, messagesFromMe chan message, deny chan string, config Config) {

	dcs := chanmap.New() // this is where the denied channels are stored, so we can close them if we get deny requests

	go func() {
		for {
			select {
			case <-closed:
				break
			case bid := <-deny:
				err := dcs.DeleteAndCloseParent(bid) //close all connections with this booking id
				if err != nil {
					log.WithFields(log.Fields{"error": err.Error(), "bid": bid}).Error("error closing connections for bid")
				}

			}
		}
	}()

	//hub := newHub() // shift this initialisation outside this function so we can share hub with access server for handling /status endpoint
	config.Hub.SetDenyChannelStore(dcs)
	go config.Hub.run()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveWs(closed, w, r, config)
	})

	var wg sync.WaitGroup
	wg.Add(1)

	go statsClient(closed, &wg, config)

	addr := ":" + strconv.Itoa(config.Listen)

	h := &http.Server{Addr: addr, Handler: nil}

	go func() {
		if err := h.ListenAndServe(); err != nil {
			log.Errorf("ListenAndServe: %s ", err.Error()) //TODO upgrade to fatal once httptest is supported
		}
	}()

	<-closed

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := h.Shutdown(ctx)
	if err != nil {
		log.Errorf("ListenAndServe.Shutdown(): %s", err.Error())
	}

	wg.Wait()
	parentwg.Done()
	log.Debug("handleConnections is done")
}
