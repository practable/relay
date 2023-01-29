package permission

import (
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
)

func TestNewTokenValidate(t *testing.T) {

	audience := "some.host.io"
	topic := "someid"
	scopes := []string{"read", "write"}
	nbf := time.Now().Unix()
	iat := nbf
	exp := nbf + 10
	ct := "session"

	token := NewToken(audience, ct, topic, scopes, iat, nbf, exp)

	assert.Equal(t, audience, token.Audience[0])
	assert.Equal(t, ct, token.ConnectionType)
	assert.Equal(t, topic, token.Topic)
	assert.Equal(t, scopes, token.Scopes)
	assert.Equal(t, *jwt.NewNumericDate(time.Unix(iat, 0)), *token.IssuedAt)
	assert.Equal(t, *jwt.NewNumericDate(time.Unix(nbf, 0)), *token.NotBefore)
	assert.Equal(t, *jwt.NewNumericDate(time.Unix(exp, 0)), *token.ExpiresAt)

	assert.True(t, HasRequiredClaims(token))

}
