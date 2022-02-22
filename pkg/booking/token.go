package booking

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/timdrysdale/relay/pkg/permission"
)

// Token returns a signed JWT token
func Token(audience, ct, topic, secret string, scopes []string, iat, nbf, exp int64) (string, error) {

	var claims permission.Token
	claims.IssuedAt = jwt.NewNumericDate(time.Unix(iat, 0))
	claims.NotBefore = jwt.NewNumericDate(time.Unix(nbf, 0))
	claims.ExpiresAt = jwt.NewNumericDate(time.Unix(exp, 0))
	claims.Audience = jwt.ClaimStrings{audience}
	claims.Topic = topic
	claims.ConnectionType = ct // e.g. shell
	claims.Scopes = scopes     // e.g. "host", "client", or "stats"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))

}
