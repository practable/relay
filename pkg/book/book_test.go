package book

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
	"github.com/timdrysdale/relay/pkg/booking"
	"github.com/timdrysdale/relay/pkg/booking/models"
	lit "github.com/timdrysdale/relay/pkg/login"
	"github.com/timdrysdale/relay/pkg/pool"
)

var ps *pool.PoolStore
var host, secret string
var bookingDuration int64

func TestMain(m *testing.M) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	secret = "somesecret"
	bookingDuration = int64(180)

	ps = pool.NewPoolStore().
		WithSecret(secret).
		WithBookingTokenDuration(bookingDuration)

	port, err := freeport.GetFreePort()
	if err != nil {
		panic(err)
	}

	host = "http://[::]:" + strconv.Itoa(port)

	go booking.API(ctx, port, host, secret, ps)

	time.Sleep(time.Second)

	exitVal := m.Run()

	os.Exit(exitVal)
}

func TestBooking(t *testing.T) {

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

	btr := &models.Bookingtoken{}

	err = json.Unmarshal(body, btr)

	assert.NoError(t, err)

	bookingTokenReturned := *(btr.Token)

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

	btr = &models.Bookingtoken{}

	err = json.Unmarshal(body, btr)
	assert.NoError(t, err)

	token, err = jwt.ParseWithClaims(*(btr.Token), &lit.Token{}, func(token *jwt.Token) (interface{}, error) {
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

func TestGetGroupIDByName(t *testing.T) {

	g0 := pool.NewGroup("stuff")
	ps.AddGroup(g0)
	defer ps.DeleteGroup(g0)

	g1 := pool.NewGroup("things")
	ps.AddGroup(g1)
	defer ps.DeleteGroup(g1)

	claims := &lit.Token{}
	claims.Audience = host
	claims.Groups = []string{"stuff"}
	claims.Scopes = []string{"booking", "user"}
	claims.IssuedAt = ps.GetTime() - 1
	claims.NotBefore = ps.GetTime() - 1
	claims.ExpiresAt = claims.NotBefore + ps.BookingTokenDuration

	// sign user token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Sign and get the complete encoded token as a string using the secret
	bearer, err := token.SignedString([]byte(ps.Secret))
	assert.NoError(t, err)

	client := &http.Client{}
	req, err := http.NewRequest("GET", host+"/api/v1/groups", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)
	q := req.URL.Query()
	q.Add("name", "stuff")
	req.URL.RawQuery = q.Encode()
	resp, err := client.Do(req)
	assert.NoError(t, err)
	body, _ := ioutil.ReadAll(resp.Body)

	ids := []string{}
	err = json.Unmarshal(body, &ids)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ids))
	assert.Equal(t, g0.ID, ids[0])

	// check request fails if group not in groups
	req, err = http.NewRequest("GET", host+"/api/v1/groups", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)
	q = req.URL.Query()
	q.Add("name", "things")
	req.URL.RawQuery = q.Encode()
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, _ = ioutil.ReadAll(resp.Body)
	assert.Equal(t, "\"Missing Group in Groups Claim\"\n", string(body))

}
