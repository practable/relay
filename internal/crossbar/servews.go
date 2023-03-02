package crossbar

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/eclesh/welford"
	"github.com/google/uuid"
	"github.com/practable/relay/internal/permission"
	"github.com/practable/relay/internal/util"
	log "github.com/sirupsen/logrus"
)

// ConnectionType represents whether the connection is session, shell, or unsupported
type ConnectionType int

// Session, Shell and Unsupported are enumerated values of ConnectionType
const (
	Session ConnectionType = iota
	Shell
	Unsupported
)

// serveWs handles websocket requests from clients.
func serveWs(closed <-chan struct{}, hub *Hub, w http.ResponseWriter, r *http.Request, config Config) {

	// check if topic is of a supported type before we go any further
	ct := Unsupported

	path := slashify(r.URL.Path)

	log.WithField("path", path).Trace()

	prefix := getConnectionTypeFromPath(path)
	topic := getTopicFromPath(path)

	log.Debug(fmt.Sprintf("%s -> %s and %s\n", path, prefix, topic))

	if prefix == "session" {
		ct = Session
	} else {
		ct = Unsupported
	}

	log.WithField("connectionType", ct).Trace()

	if ct == Unsupported {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		log.WithField("connectionType", prefix).Error("connection_type unsupported")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithFields(log.Fields{"topic": topic, "error": err.Error()}).Error("failed to upgrade to websocket")
		return
	}

	log.Trace("upgraded to ws") //Cannot return any http responses from here on

	// Enforce permissions by exchanging the authcode for a connection ticket
	// which contains expiry time, route, and permissions

	// Get the first code query param, lowercase only

	code := r.URL.Query().Get("code")

	log.WithField("code", code).Debug()

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
	// It's already been validated, so a claims check is all we need to do here
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

	if token.ConnectionType != "session" {
		log.WithFields(log.Fields{"topic": topic, "booking_id": token.BookingID, "connection_type": prefix}).Error("unauthorized because connection_type is not a session")
		return
	}

	// initialise statistics
	tx := &Frames{size: welford.New(), ns: welford.New(), mu: &sync.RWMutex{}}
	rx := &Frames{size: welford.New(), ns: welford.New(), mu: &sync.RWMutex{}}
	exp := time.Unix((*token.ExpiresAt).Unix(), 0) // jwt.NumericDate underlying type is time.Time
	stats := &Stats{connectedAt: time.Now(), expiresAt: exp, tx: tx, rx: rx}

	denied := make(chan struct{})

	client := &Client{hub: hub,
		bookingID:  token.BookingID,
		conn:       conn,
		denied:     denied,
		expiresAt:  (*token.ExpiresAt).Unix(),
		send:       make(chan message, config.BufferSize),
		topic:      topic,
		stats:      stats,
		name:       uuid.New().String(),
		userAgent:  r.UserAgent(),
		remoteAddr: r.Header.Get("X-Forwarded-For"),
		audience:   config.Audience,
		canRead:    canRead,
		canWrite:   canWrite,
	}
	client.hub.register <- client

	cf := log.Fields{
		"booking_id":  token.BookingID,
		"expires_at":  (*token.ExpiresAt).Unix(),
		"topic":       topic,
		"stats":       true,
		"buffer_size": config.BufferSize,
		"name":        uuid.New().String(),
		"user_agent":  r.UserAgent(),
		"remote_addr": r.Header.Get("X-Forwarded-For"),
		"audience":    config.Audience,
		"can_read":    canRead,
		"can_write":   canWrite,
	}

	log.WithFields(cf).Infof("new connection")

	cancelled := make(chan struct{})

	// cancel the connection when the token has expired or when session is curtailed
	go func() {

		select {
		case <-time.After(time.Duration(ttl) * time.Second): //prevents leakage if denied, as token will eventually expire, ending this goro
			log.WithFields(cf).WithField("reason", "token expired").Info("connection closed")
		case <-denied:
			log.WithFields(cf).WithField("reason", "token denied").Info("connection closed")
		}
		// don't include a case for closed here, because we needn't log an individual connection shutting down
		// if adding case <- closed: then check closed and cancelled being set at same time is handled in writePump without a hang that would slow shutdown
		close(cancelled)
	}()

	go client.writePump(closed, cancelled)
	go client.readPump()
	return

}
