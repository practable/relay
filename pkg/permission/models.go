package permission

import (
	"github.com/dgrijalva/jwt-go"
)

// Permission represents claims required in the apiKey JWT
// When specifying a stream to pkg/booking, the required fields are
// - Topic
// - ConnectionType
// - Scopes
// - Audience

type Token struct {

	// Topic identifies the communication channel;
	// think of it as a session_id, or room (webrtc)
	// Don't reuse standard claim Subject as that is for a
	// usually for a user and is reserved for later usage.
	Topic string `json:"topic"`

	// Determines the type of communication channel
	// and hence required scopes + hub/client implementation
	// e.g. "session" or "shell"
	ConnectionType string `json:"prefix"`

	// Scopes controlling access to relay;
	// either ["read"],["write"], or ["read","write"] for session, or ["host"]/["client"] for shell
	Scopes []string `json:"scopes"`

	// ConnectionIDSalt remains hidden within a relay
	// and is used to obscure the actual topic used in a hub
	// from visibility in access logs e.g. in shellbar
	TopicSalt string `json:"topicSalt,omitempty"`

	// AlertHost controls whether making _this_
	// particular connection should alert the host
	// This is needed for ssh hosts in shellbar
	AlertHost bool `json:"alertHost,omitempty"`

	jwt.StandardClaims
}

func NewToken(audience, connectionType, topic string, scopes []string, iat, nbf, exp int64) Token {
	return Token{
		Topic:          topic,
		Scopes:         scopes,
		ConnectionType: connectionType,
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  iat,
			NotBefore: nbf,
			ExpiresAt: exp,
			Audience:  audience,
		},
	}
}

func SetTopicSalt(token *Token, salt string) {
	token.TopicSalt = salt
}

func SetAlertHost(token *Token, alertHost bool) {
	token.AlertHost = alertHost
}

func HasRequiredClaims(token Token) bool {

	if token.Topic == "" ||
		len(token.Scopes) == 0 ||
		token.ConnectionType == "" ||
		token.Audience == "" ||
		token.ExpiresAt == 0 {
		return false
	}
	return true
}
