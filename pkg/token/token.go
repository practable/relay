package token

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/practable/relay/internal/permission"
)

// New returns a signed JWT token
func New(iat, nbf, exp time.Time, scopes []string, aud, bid, connectionType, secret, topic string) (string, error) {

	var claims permission.Token

	claims.IssuedAt = jwt.NewNumericDate(iat)
	claims.NotBefore = jwt.NewNumericDate(nbf)
	claims.ExpiresAt = jwt.NewNumericDate(iat)

	claims.Audience = jwt.ClaimStrings{aud}
	claims.BookingID = bid
	claims.Topic = topic
	claims.ConnectionType = connectionType
	claims.Scopes = scopes

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign and return the complete encoded token as a string using the secret
	return token.SignedString([]byte(secret))

}
