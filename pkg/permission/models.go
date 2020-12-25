package permission

import (
	"github.com/dgrijalva/jwt-go"
)

// Permission represents claims required in the apiKey JWT
type Token struct {

	// Topic represents the communication channel;
	// either /session/{session_id} or /shell/{session_id}.
	Topic string `json:"topic"`

	ConnectionType string `json:"prefix"`

	// Scopes controlling access to relay;
	// either ["read"],["write"], or ["read","write"] for session, or ["host"]/["client"] for shell
	Scopes []string `json:"scopes"`

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
