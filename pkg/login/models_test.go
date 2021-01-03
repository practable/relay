package login

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTokenValidate(t *testing.T) {

	audience := "some.host.io"
	scopes := []string{"login", "user"}
	groups := []string{"somecourse", "everyone"}
	nbf := time.Now().Unix()
	iat := nbf
	exp := nbf + 10

	token := NewToken(audience, groups, scopes, iat, nbf, exp)
	assert.Equal(t, audience, token.Audience)
	assert.Equal(t, groups, token.Groups)
	assert.Equal(t, scopes, token.Scopes)
	assert.Equal(t, iat, token.IssuedAt)
	assert.Equal(t, nbf, token.NotBefore)
	assert.Equal(t, exp, token.ExpiresAt)

	assert.True(t, HasRequiredClaims(token))

	SetSubject(&token, "someuser")

	assert.Equal(t, "someuser", token.Subject)

}
