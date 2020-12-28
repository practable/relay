package permission

import (
	"testing"
	"time"

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

	assert.Equal(t, audience, token.Audience)
	assert.Equal(t, ct, token.ConnectionType)
	assert.Equal(t, topic, token.Topic)
	assert.Equal(t, scopes, token.Scopes)
	assert.Equal(t, iat, token.IssuedAt)
	assert.Equal(t, nbf, token.NotBefore)
	assert.Equal(t, exp, token.ExpiresAt)

	assert.True(t, HasRequiredClaims(token))

}

func TestAddTopicSaltAlertHost(t *testing.T) {

	audience := "some.host.io"
	topic := "someid"
	scopes := []string{"read", "write"}
	nbf := time.Now().Unix()
	iat := nbf
	exp := nbf + 10
	ct := "session"

	token := NewToken(audience, ct, topic, scopes, iat, nbf, exp)

	assert.Equal(t, "", token.TopicSalt)

	SetTopicSalt(&token, "somesalt")

	assert.Equal(t, "somesalt", token.TopicSalt)

	assert.Equal(t, false, token.AlertHost)

	SetAlertHost(&token, true)

	assert.Equal(t, true, token.AlertHost)

	SetAlertHost(&token, false)

	assert.Equal(t, false, token.AlertHost)

}
