package crossbar

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/eclesh/welford"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/practable/relay/internal/permission"
	"github.com/practable/relay/internal/util"
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

	log.Trace(fmt.Sprintf("%s -> %s and %s\n", path, prefix, topic))

	if prefix == "session" {
		ct = Session
	}
	if prefix == "shell" {
		ct = Unsupported //TODO implement shell!
	}

	log.WithField("connectionType", ct).Trace()

	if ct == Unsupported {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		log.WithField("connectionType", prefix).Error("connectionType unsuported")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithField("error", err).Error("serveWs failed to upgrade to websocket")
		return
	}

	log.Trace("upgraded to ws") //Cannot return any http responses from here on

	// Enforce permissions by exchanging the authcode for a connection ticket
	// which contains expiry time, route, and permissions

	// Get the first code query param, lowercase only

	code := r.URL.Query().Get("code")

	log.WithField("code", code).Trace()

	// if no code or empty, return 401
	if code == "" {
		log.WithField("topic", topic).Info("Unauthorized - No Code")
		return
	}

	// Exchange code for token

	token, err := config.CodeStore.ExchangeCode(code)

	log.WithFields(log.Fields{"token": util.Compact(token), "error": err}).Trace("Exchange code")

	if err != nil {
		log.WithField("topic", topic).Info("Unauthorized - Invalid Code")
		return
	}

	// check token is a permission token so we can process it properly
	// It's been validated so we don't need to re-do that
	if !permission.HasRequiredClaims(token) {
		log.WithField("topic", topic).Info("Unauthorized - original token missing claims")
		return
	}

	now := config.CodeStore.GetTime()

	if token.NotBefore.After(time.Unix(now, 0)) {
		log.WithField("topic", topic).Info("Unauthorized - Too early")
		return
	}

	ttl := token.ExpiresAt.Unix() - now

	log.WithFields(log.Fields{"ttl": ttl, "topic": topic}).Trace()

	audok := false

	for _, aud := range token.Audience {
		if aud == config.Audience {
			audok = true
		}
	}

	topicBad := (topic != token.Topic)
	expired := ttl < 0

	if (!audok) || topicBad || expired {
		log.WithFields(log.Fields{"audienceOK": audok, "topicOK": !topicBad, "expired": expired, "topic": topic}).Trace("Token invalid")
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
		log.WithFields(log.Fields{"topic": topic, "scopes": token.Scopes}).Trace("No valid scopes")
		return
	}

	cancelled := make(chan struct{})

	// cancel the connection when the token has expired
	go func() {
		time.Sleep(time.Duration(ttl) * time.Second)
		close(cancelled)
	}()

	if ct == Session {
		// initialise statistics
		tx := &Frames{size: welford.New(), ns: welford.New(), mu: &sync.RWMutex{}}
		rx := &Frames{size: welford.New(), ns: welford.New(), mu: &sync.RWMutex{}}
		stats := &Stats{connectedAt: time.Now(), tx: tx, rx: rx}

		client := &Client{hub: hub,
			conn:       conn,
			send:       make(chan message, 256),
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

		go client.writePump(closed, cancelled)
		go client.readPump()
		return
	}

}
