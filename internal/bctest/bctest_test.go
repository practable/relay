package bctest

import (
	"context"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/golang-jwt/jwt/v4"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
	apiclient "github.com/practable/relay/internal/bc/client"
	login "github.com/practable/relay/internal/bc/client/login"
	"github.com/practable/relay/internal/booking"
	"github.com/practable/relay/internal/bookingstore"
	lit "github.com/practable/relay/internal/login"
	"github.com/practable/relay/internal/pool"
	"github.com/xtgo/uuid"
)

var l *bookingstore.Limit
var ps *pool.Store
var useLocal bool
var adminBearer, userBearer string
var secret string

func init() {

	useLocal = true

	SetDebug(false)
}

func TestMain(m *testing.M) {

	var lt lit.Token
	var cfg *apiclient.TransportConfig
	var adminLoginBearer, userLoginBearer string

	iat := time.Now().Unix() - 1
	nbf := iat
	exp := nbf + 30

	if useLocal {

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		secret = "somesecret"

		ps = pool.NewStore().
			WithSecret(secret).
			WithBookingTokenDuration(int64(180))

		l = bookingstore.New(ctx).WithFlush(time.Minute).WithMax(2).WithProvisionalPeriod(5 * time.Second)

		port, err := freeport.GetFreePort()
		if err != nil {
			panic(err)
		}

		host := "[::]:" + strconv.Itoa(port)

		audience := "http://" + host

		go booking.API(ctx, port, audience, secret, ps, l)

		time.Sleep(time.Second)

		lt = lit.NewToken(audience, []string{"everyone"}, []string{}, []string{"login:admin"}, iat, nbf, exp)
		adminLoginBearer, err = lit.Signed(lt, secret)
		if err != nil {
			panic(err)
		}

		lt = lit.NewToken(audience, []string{"everyone"}, []string{}, []string{"login:user"}, iat, nbf, exp)
		userLoginBearer, err = lit.Signed(lt, secret)
		if err != nil {
			panic(err)
		}

		cfg = apiclient.DefaultTransportConfig().WithHost(host).WithSchemes([]string{"http"})

	} else { //remote

		remoteSecret, err := ioutil.ReadFile("../../secret/book.practable.io.secret")
		if err != nil {
			panic(err)
		}

		secret = strings.TrimSuffix(string(remoteSecret), "\n")

		lt = lit.NewToken("https://book.practable.io", []string{"everyone"}, []string{}, []string{"login:admin"}, iat, nbf, exp)
		adminLoginBearer, err = lit.Signed(lt, secret)
		if err != nil {
			panic(err)
		}

		lt = lit.NewToken("https://book.practable.io", []string{"everyone"}, []string{}, []string{"login:user"}, iat, nbf, exp)
		userLoginBearer, err = lit.Signed(lt, secret)
		if err != nil {
			panic(err)
		}

		cfg = apiclient.DefaultTransportConfig().WithSchemes([]string{"https"})

	}

	// admin login
	loginAuth := httptransport.APIKeyAuth("Authorization", "header", adminLoginBearer)

	bc := apiclient.NewHTTPClientWithConfig(nil, cfg)

	timeout := 10 * time.Second

	params := login.NewLoginParams().WithTimeout(timeout)

	resp, err := bc.Login.Login(params, loginAuth)

	if err != nil {
		panic(err)
	}

	adminBearer = *resp.GetPayload().Token

	// user login
	loginAuth = httptransport.APIKeyAuth("Authorization", "header", userLoginBearer)

	params = login.NewLoginParams().WithTimeout(timeout)

	resp, err = bc.Login.Login(params, loginAuth)

	if err != nil {
		panic(err)
	}

	userBearer = *resp.GetPayload().Token

	exitVal := m.Run()

	os.Exit(exitVal)
}

func TestBearers(t *testing.T) {

	// admin
	token, err := jwt.ParseWithClaims(adminBearer, &lit.Token{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)

	claims, ok := token.Claims.(*lit.Token)

	assert.True(t, ok)
	assert.True(t, token.Valid)

	assert.Equal(t, []string{"everyone"}, claims.Groups)
	assert.Equal(t, []string{"booking:admin"}, claims.Scopes)
	assert.True(t, claims.ExpiresAt.After(time.Now().Add(30*time.Second)))

	_, err = uuid.Parse(claims.Subject)
	assert.NoError(t, err)

	// user
	token, err = jwt.ParseWithClaims(userBearer, &lit.Token{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)

	claims, ok = token.Claims.(*lit.Token)

	assert.True(t, ok)
	assert.True(t, token.Valid)

	assert.Equal(t, []string{"everyone"}, claims.Groups)
	assert.Equal(t, []string{"booking:user"}, claims.Scopes)
	assert.True(t, claims.ExpiresAt.After(time.Now().Add(30*time.Second)))
	_, err = uuid.Parse(claims.Subject)
	assert.NoError(t, err)

}
