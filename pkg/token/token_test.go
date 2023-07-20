package token

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {

	iat := time.Unix(1658306108, 0)
	nbf := time.Unix(1658307108, 0)
	exp := time.Unix(1658308108, 0)
	scopes := []string{"read", "write"}
	aud := "https://example.com"
	bid := "xxx000"
	connectionType := "session"
	secret := "some_secret"
	topic := "expt00"

	token, err := New(iat, nbf, exp, scopes, aud, bid, connectionType, secret, topic)

	expected := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJib29raW5nX2lkIjoieHh4MDAwIiwidG9waWMiOiJleHB0MDAiLCJwcmVmaXgiOiJzZXNzaW9uIiwic2NvcGVzIjpbInJlYWQiLCJ3cml0ZSJdLCJhdWQiOlsiaHR0cHM6Ly9leGFtcGxlLmNvbSJdLCJleHAiOjE2NTgzMDgxMDgsIm5iZiI6MTY1ODMwNzEwOCwiaWF0IjoxNjU4MzA2MTA4fQ.H93xnfopLhc8UGLqFsX5foP2MPmZohb8WtuojbP3ORQ"

	assert.NoError(t, err)
	assert.Equal(t, expected, token)

}
