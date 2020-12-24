package crossbar

import (
	"net/http"
	"strings"
	"time"

	"github.com/eclesh/welford"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/relay/pkg/permission"
	"github.com/timdrysdale/relay/pkg/util"
)

type ConnectionType int

const (
	Session ConnectionType = iota
	Shell
	Unsupported
)

// serveWs handles websocket requests from clients.
func serveWs(closed <-chan struct{}, hub *Hub, w http.ResponseWriter, r *http.Request, config Config) {

	// check if topic is of a supported type before we go any further
	ct := Unsupported

	topic := slashify(r.URL.Path)

	log.WithField("topic", topic).Trace()

	if strings.HasPrefix(topic, "/session/") {
		ct = Session
	}
	if strings.HasPrefix(topic, "/shell/") {
		ct = Unsupported //TODO implement shell!
	}

	log.WithField("connectionType", ct).Trace()

	if ct == Unsupported {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		log.WithField("topic", topic).Error("topic unsuported")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.WithField("error", err).Error("serveWs failed to upgrade to websocket")
		return
	}

	log.Trace("upgraded to ws") //Cannot return any http responses from here on

	// Enforce permissions by exchanging the authcode for a connection ticket
	// which contains expiry time, route, and permissions

	// Get the first code query param, lowercase only
	var code string

	code = r.URL.Query().Get("code")

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
	if !permission.ValidPermissionToken(token) {
		log.WithField("topic", topic).Info("Unauthorized - Not a permission token")
		return
	}

	// convert claims into permission.Token for convenience

	p, err := permission.GetPermissionToken(token)

	if err != nil {
		log.WithField("topic", topic).Info("Unauthorized - Not a permission token")
		return
	}

	now := config.CodeStore.GetTime()

	if p.NotBefore > now {
		log.WithField("topic", topic).Info("Unauthorized - Too early")
		return
	}

	ttl := p.ExpiresAt - now

	log.WithFields(log.Fields{"ttl": ttl, "topic": topic}).Trace()

	audienceBad := (config.Audience != p.Audience)
	topicBad := (topic != p.Topic)
	expired := ttl < 0

	if audienceBad || topicBad || expired {
		log.WithFields(log.Fields{"audienceBad": audienceBad, "topicBad": topicBad, "expired": expired, "topic": topic}).Trace("Token invalid")
		return
	}

	// check permissions

	var canRead, canWrite bool

	for _, scope := range p.Scopes {
		if scope == "read" {
			canRead = true
		}
		if scope == "write" {
			canWrite = true
		}
	}

	if !(canRead || canWrite) {
		log.WithFields(log.Fields{"topic": topic, "scopes": p.Scopes}).Trace("No valid scopes")
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
		tx := &Frames{size: welford.New(), ns: welford.New()}
		rx := &Frames{size: welford.New(), ns: welford.New()}
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
