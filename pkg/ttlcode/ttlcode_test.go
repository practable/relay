package ttlcode_test

import (
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
	"github.com/timdrysdale/relay/pkg/ttlcode"
)

func createToken() jwt.Token {
	// Create a new token object, specifying signing method and the claims
	// you would like it to contain.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"foo": "bar",
		"nbf": time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
	})

	return *token
}

func createTokenWithClaim(claim, value string) jwt.Token {
	// Create a new token object, specifying signing method and the claims
	// you would like it to contain.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"foo": "bar",
		"nbf": time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
		claim: value,
	})

	return *token
}

func TestCreateTokenStoreTTL(t *testing.T) {

	c := ttlcode.NewDefaultCodeStore().
		WithTTL(60)

	assert.Equal(t, int64(60), c.GetTTL())

}

func TestSubmitExchangeToken(t *testing.T) {

	t.Parallel()

	ttl := int64(3)

	c := ttlcode.NewDefaultCodeStore().
		WithTTL(ttl)
	defer c.Close()

	earlier := ttlcode.GetTime()

	token := createToken()
	code := c.SubmitToken(token)

	assert.True(t, len(code) >= 36, "code too little entropy")

	assert.True(t, code != "")

	tok, err := c.ExchangeCode(code)

	assert.NoError(t, err, "error exchanging code")

	later := ttlcode.GetTime()

	assert.True(t, later < (earlier+ttl), "exchange took too long")

	assert.Equal(t, token, tok)

	_, err = c.ExchangeCode(code)

	evenlater := ttlcode.GetTime()

	// we want to try (+fail at) getting token a second time before it expires
	assert.True(t, evenlater < (earlier+ttl), "failed exchange took too long")

	assert.Error(t, err, "should have failed to get code twice")
}

func TestTokensExpire(t *testing.T) {

	t.Parallel()

	ttl := int64(1)

	c := ttlcode.NewDefaultCodeStore().
		WithTTL(ttl)
	defer c.Close()

	token := createTokenWithClaim("token", "99")
	code := c.SubmitToken(token)

	<-time.After(3 * time.Second)

	_, err := c.ExchangeCode(code)

	assert.Error(t, err, "token should have expired")

}

func TestStoreIsCleaned(t *testing.T) {

	t.Parallel()

	ttl := int64(1)

	c := ttlcode.NewDefaultCodeStore().
		WithTTL(ttl)
	defer c.Close()

	token := createTokenWithClaim("token", "99")

	_ = c.SubmitToken(token)

	<-time.After(1 * time.Second)

	assert.Equal(t, 1, c.GetCodeCount())

	<-time.After(2 * time.Second)

	assert.Equal(t, 0, c.GetCodeCount())

}

func TestTokensAreDistinct(t *testing.T) {

	t.Parallel()

	ttl := int64(3)

	c := ttlcode.NewDefaultCodeStore().
		WithTTL(ttl)
	defer c.Close()

	token0 := createTokenWithClaim("token", "0")
	code0 := c.SubmitToken(token0)

	token1 := createTokenWithClaim("token", "1")
	code1 := c.SubmitToken(token1)

	// reverse order
	tok1, err := c.ExchangeCode(code1)
	assert.NoError(t, err)

	tok0, err := c.ExchangeCode(code0)
	assert.NoError(t, err)

	assert.NotEqual(t, token0, token1)
	assert.NotEqual(t, code0, code1)
	assert.NotEqual(t, tok0, tok1)

	assert.Equal(t, token0, tok0)
	assert.Equal(t, token1, tok1)
}
