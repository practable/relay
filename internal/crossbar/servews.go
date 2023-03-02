package crossbar

import (
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
		// initialise statistics
		tx := &Frames{size: welford.New(), ns: welford.New(), mu: &sync.RWMutex{}}
		rx := &Frames{size: welford.New(), ns: welford.New(), mu: &sync.RWMutex{}}
		exp := time.Unix((*token.ExpiresAt).Unix(), 0) // jwt.NumericDate underlying type is time.Time
		stats := &Stats{connectedAt: time.Now(), expiresAt: exp, tx: tx, rx: rx}

		client := &Client{hub: hub,
			bookingID:  token.BookingID,
			conn:       conn,
			denied:     denied,
			expiresAt:  (*token.ExpiresAt).Unix(),
			send:       make(chan message, int(config.BufferSize)),
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
