package booking

import (
	"github.com/golang-jwt/jwt/v4"
	"github.com/timdrysdale/relay/pkg/permission"
)

// Token returns a signed JWT token
func Token(audience, ct, topic, secret string, scopes []string, iat, nbf, exp int64) (string, error) {

	var claims permission.Token
	claims.IssuedAt = iat
	claims.NotBefore = nbf
	claims.ExpiresAt = exp
	claims.Audience = audience
	claims.Topic = topic
	claims.ConnectionType = ct // e.g. shell
	claims.Scopes = scopes     // e.g. "host", "client", or "stats"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))

}
