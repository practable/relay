package permission

import (
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
)

// Permission represents claims required in the apiKey JWT
// When specifying a stream to pkg/booking, the required fields are
// - Topic
// - ConnectionType
// - Scopes
// - Audience

// Token represents a JWT token
type Token struct {

	// BookingID represents the booking id/name that the token has been generated
	// for, so that connections can be dropped / denied if a booked session
	// is cancelled
	BookingID string `json:"booking_id"`

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

	jwt.RegisteredClaims `yaml:",omitempty"`
}

// NewToken returns a Token populated with the supplied information
func NewToken(audience, connectionType, topic string, scopes []string, iat, nbf, exp int64) Token {

	return Token{
		Topic:          topic,
		Scopes:         scopes,
		ConnectionType: connectionType,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Unix(iat, 0)),
			NotBefore: jwt.NewNumericDate(time.Unix(nbf, 0)),
			ExpiresAt: jwt.NewNumericDate(time.Unix(exp, 0)),
			Audience:  []string{audience},
		},
	}
}

// SetBookingID sets the bookingID (so that connections can be cancelled if a booking is cancelled)
func (t *Token) SetBookingID(bookingID string) {
	t.BookingID = bookingID
}

// HasRequiredClaims returns false if the Token is missing any required elements
func HasRequiredClaims(token Token) bool {

	if token.Topic == "" ||
		len(token.Scopes) == 0 ||
		token.ConnectionType == "" ||
		len(token.Audience) == 0 ||
		(*token.ExpiresAt).IsZero() {
		return false
	}
	return true
}
