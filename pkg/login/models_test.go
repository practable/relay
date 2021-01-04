package login

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTokenValidate(t *testing.T) {

	audience := "some.host.io"
	scopes := []string{"login", "user"}
	groups := []string{"somecourse", "everyone"}
	nbf := int64(1609721410)
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

	pretty := `{
	"groups": [
		"somecourse",
		"everyone"
	],
	"scopes": [
		"login",
		"user"
	],
	"aud": "some.host.io",
	"exp": 1609721420,
	"iat": 1609721410,
	"nbf": 1609721410,
	"sub": "someuser"
}`

	assert.Equal(t, pretty, token.String())
}
