package shellbar

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/eclesh/welford"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/relay/pkg/permission"
)

type ConnectionType int

const (
	Session ConnectionType = iota
	Shell
	Unsupported
)

// serveWs handles websocket requests from clients.
func serveWs(closed <-chan struct{}, hub *Hub, w http.ResponseWriter, r *http.Request, config Config) {

	id := "shellbar.serveWs(" + uuid.New().String()[0:6] + ")"

	// check if topic is of a supported type before we go any further
	ct := Unsupported

	path := slashify(r.URL.Path)

	connectionType := getConnectionTypeFromPath(path)
	topic := getTopicFromPath(path)

	if connectionType == "shell" {
		ct = Shell
	}

	if ct == Unsupported {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		log.WithField("connectionType", connectionType).Errorf("%s: connectionType %s unsupported", id, connectionType)
		return
	}

	log.WithFields(log.Fields{"path": r.URL.Path}).Infof("%s: received %s connection to topic %s at %s", id, connectionType, topic, r.URL.Path)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithFields(log.Fields{"path": r.URL.Path, "error": err}).Errorf("%s: failed to upgrade to websocket at %s", id, r.URL.Path)
		return
	}

	//Cannot return any http responses from here on

	// Enforce permissions by exchanging the authcode for a connection ticket
	// which contains expiry time, route, and permissions

	// Get the first code query param, lowercase only
	var code string

	code = r.URL.Query().Get("code")

	// if no code or empty, return 401
	if code == "" {
		log.WithField("url", r.URL.String()).Infof("%s: Unauthorized - No Code in %s", id, r.URL.String())
		return
	}

	// Exchange code for token

	token, err := config.CodeStore.ExchangeCode(code)

	if err != nil {
		log.WithFields(log.Fields{"topic": topic, "error": err}).Infof("%s: Unauthorized - Invalid Code because %s", id, err.Error())
		return
	}

	// check token is a permission token so we can process it properly
	// It's been validated so we don't need to re-do that
	if !permission.HasRequiredClaims(token) {
		log.WithField("topic", topic).Infof("%s: Unauthorized - original token missing claims", id)
		return
	}

	now := config.CodeStore.GetTime()

	if token.NotBefore > now {
		log.WithField("topic", topic).Infof("%s: Unauthorized - Too early", id)
		return
	}

	ttl := token.ExpiresAt - now

	audienceBad := (config.Audience != token.Audience)
	topicBad := (topic != token.Topic)
	expired := ttl < 0

	if audienceBad || topicBad || expired {
		log.WithFields(log.Fields{"audienceBad": audienceBad, "topicBad": topicBad, "expired": expired, "topic": topic}).Tracef("%s: Token invalid", id)
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
		log.WithFields(log.Fields{"topic": topic, "scopes": token.Scopes}).Tracef("%s: No valid scopes", id)
		return
	}

	cancelled := make(chan struct{})

	// cancel the connection when the token has expired
	go func() {
		time.Sleep(time.Duration(ttl) * time.Second)
		close(cancelled)
	}()

	if ct == Shell {
		// initialise statistics
		tx := &Frames{size: welford.New(), ns: welford.New()}
		rx := &Frames{size: welford.New(), ns: welford.New()}
		stats := &Stats{connectedAt: time.Now(), tx: tx, rx: rx}

		client := &Client{hub: hub,
			conn:          conn,
			send:          make(chan message, 256),
			topic:         topic + token.TopicSalt,
			stats:         stats,
			name:          uuid.New().String(),
			userAgent:     r.UserAgent(),
			remoteAddr:    r.Header.Get("X-Forwarded-For"),
			audience:      config.Audience,
			canRead:       canRead,
			canWrite:      canWrite,
			hostAlertUUID: uuid.New().String(),
		}
		client.hub.register <- client

		log.WithField("Topic", client.topic).Tracef("%s: registering client at topic %s with name %s", id, client.topic, client.name)

		go client.writePump(closed, cancelled)
		go client.readPump()

		log.WithField("topic", topic+token.TopicSalt).Tracef("%s: started shellrelay client on topic %s", id, topic+token.TopicSalt)

		if token.AlertHost {
			log.WithField("topic", topic+token.TopicSalt).Tracef("%s: alert host of topic %s to new client %s with salt %s", id, topic, client.name, token.TopicSalt)

			// alert SSH host agent to make a new connection to relay at the same address
			// no stats required because we are not registering to receive messages
			adminClient := &Client{
				topic: getHostTopicFromUniqueTopic(topic),
				name:  uuid.New().String(),
			}

			permission.SetAlertHost(&token, false) //turn off host alert
			code = config.CodeStore.SubmitToken(token)

			if code == "" {
				log.Errorf("%s: failed to submit host connect token in exchange for a code", id)
				return
			}

			// same URL as client used, but different code (and leave out the salt)
			hostAlertURI := token.Audience + "/" + token.ConnectionType + "/" + token.Topic + "?code=" + code
			ca := ConnectionAction{
				Action: "connect",
				URI:    hostAlertURI,
				UUID:   client.hostAlertUUID,
			}

			camsg, err := json.Marshal(ca)

			if err != nil {
				log.WithFields(log.Fields{"uuid": client.hostAlertUUID, "uri": hostAlertURI, "error": err}).Errorf("%s: Failed to make connectionAction message", id)
				return
			}

			hub.broadcast <- message{sender: *adminClient, data: camsg, mt: websocket.TextMessage}
			log.WithFields(log.Fields{"uuid": client.hostAlertUUID, "uri": hostAlertURI, "code": code}).Debugf("%s: sent host CONNECT for topic %s with UUID:%s at URI:%s", id, topic, client.hostAlertUUID, hostAlertURI)

		}

		return
	}

}
