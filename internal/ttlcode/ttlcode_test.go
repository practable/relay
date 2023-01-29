package ttlcode_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/practable/relay/internal/permission"
	"github.com/practable/relay/internal/ttlcode"
	"github.com/stretchr/testify/assert"
)

func createToken() permission.Token {
	// test token
	return permission.NewToken(
		"wss://some.server.io",
		"session",
		uuid.New().String(),
		[]string{"read", "write"},
		time.Now().Unix()-1,
		time.Now().Unix()-1,
		time.Now().Unix()+5,
	)
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

	assert.True(t, later < (earlier+ttl), "test issue, exchange took too long")

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

	token := createToken()

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

	token := createToken()

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

	token0 := createToken()
	code0 := c.SubmitToken(token0)

	token1 := createToken()
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

func TestDeleteByBookingID(t *testing.T) {

	t.Parallel()

	ttl := int64(3)

	c := ttlcode.NewDefaultCodeStore().
		WithTTL(ttl)
	defer c.Close()

	token0 := createToken()
	token0.BookingID = "bid0"
	code0 := c.SubmitToken(token0)

	token1 := createToken()
	token1.BookingID = "bid1"
	code1 := c.SubmitToken(token1)

	c.DeleteByBookingID("bid1")

	tok0, err := c.ExchangeCode(code0)
	assert.NoError(t, err)
	assert.Equal(t, token0, tok0)

	tok1, err := c.ExchangeCode(code1) //should be gone
	assert.Error(t, err)
	assert.Equal(t, "invalid code", err.Error())
	assert.Equal(t, permission.Token{}, tok1)
}
