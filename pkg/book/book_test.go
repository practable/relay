package book

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
	"github.com/timdrysdale/relay/pkg/booking"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/login"
	lit "github.com/timdrysdale/relay/pkg/login"
	"github.com/timdrysdale/relay/pkg/pool"
)

func TestLogin(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	secret := "somesecret"
	bookingDuration := int64(180)

	ps := pool.NewPoolStore().
		WithSecret(secret).
		WithBookingTokenDuration(bookingDuration)

	port, err := freeport.GetFreePort()
	assert.NoError(t, err)

	host := "http://[::]:" + strconv.Itoa(port)

	go booking.API(ctx, port, host, secret, ps)

	time.Sleep(time.Second)

	// start tests

	loginClaims := &lit.Token{}
	loginClaims.Audience = host
	loginClaims.Groups = []string{"somecourse", "everyone"}
	loginClaims.Scopes = []string{"login", "user"}
	loginClaims.IssuedAt = ps.GetTime() - 1
	loginClaims.NotBefore = ps.GetTime() - 1
	loginClaims.ExpiresAt = loginClaims.NotBefore + ps.BookingTokenDuration
	// sign user token
	loginToken := jwt.NewWithClaims(jwt.SigningMethodHS256, loginClaims)
	// Sign and get the complete encoded token as a string using the secret
	loginBearer, err := loginToken.SignedString([]byte(ps.Secret))
	assert.NoError(t, err)

	client := &http.Client{}

	req, err := http.NewRequest("POST", host+"/api/v1/login", nil)
	assert.NoError(t, err)
	resp, err := client.Do(req)
	assert.NoError(t, err)

	body, _ := ioutil.ReadAll(resp.Body)
	bodyStr := string([]byte(body))
	assert.Equal(t, `{"code":401,"message":"unauthenticated for invalid credentials"}`, bodyStr)

	req, err = http.NewRequest("POST", host+"/api/v1/login", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", loginBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, _ = ioutil.ReadAll(resp.Body)

	btr := &login.LoginOKBody{}

	err = json.Unmarshal(body, btr)

	assert.NoError(t, err)

	bookingTokenReturned := btr.Token

	token, err := jwt.ParseWithClaims(bookingTokenReturned, &lit.Token{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)

	claims, ok := token.Claims.(*lit.Token)

	assert.True(t, ok)
	assert.True(t, token.Valid)

	assert.Equal(t, []string{"somecourse", "everyone"}, claims.Groups)
	assert.Equal(t, []string{"booking", "user"}, claims.Scopes)
	assert.True(t, claims.ExpiresAt < time.Now().Unix()+bookingDuration+15)
	assert.True(t, claims.ExpiresAt > time.Now().Unix()+bookingDuration-15)
	assert.True(t, len(claims.Subject) >= 35)

	subject := claims.Subject //save for next test

	// Now login again with booking token in body and see that subject is retained
	respBody, err := json.Marshal(lit.TokenInBody{Token: bookingTokenReturned})
	assert.NoError(t, err)
	req, err = http.NewRequest("POST", host+"/api/v1/login", bytes.NewBuffer(respBody))
	assert.NoError(t, err)
	req.Header.Set("Content-type", "application/json")
	req.Header.Add("Authorization", loginBearer)
	resp, err = client.Do(req)

	assert.NoError(t, err)
	body, _ = ioutil.ReadAll(resp.Body)

	btr = &login.LoginOKBody{}

	err = json.Unmarshal(body, btr)
	assert.NoError(t, err)

	token, err = jwt.ParseWithClaims(btr.Token, &lit.Token{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)

	claims, ok = token.Claims.(*lit.Token)

	assert.True(t, ok)
	assert.True(t, token.Valid)

	assert.Equal(t, []string{"somecourse", "everyone"}, claims.Groups)
	assert.Equal(t, []string{"booking", "user"}, claims.Scopes)
	assert.True(t, claims.ExpiresAt < time.Now().Unix()+bookingDuration+15)
	assert.True(t, claims.ExpiresAt > time.Now().Unix()+bookingDuration-15)
	assert.True(t, len(claims.Subject) >= 35)

	// key test
	assert.Equal(t, subject, claims.Subject)

}
