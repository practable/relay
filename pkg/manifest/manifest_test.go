package manifest

import (
	"bufio"
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/phayes/freeport"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	apiclient "github.com/timdrysdale/relay/pkg/bc/client"
	login "github.com/timdrysdale/relay/pkg/bc/client/login"
	"github.com/timdrysdale/relay/pkg/booking"
	"github.com/timdrysdale/relay/pkg/bookingstore"
	lit "github.com/timdrysdale/relay/pkg/login"
	"github.com/timdrysdale/relay/pkg/pool"
	"github.com/xtgo/uuid"
	"gopkg.in/yaml.v2"
)

var debug bool
var l *bookingstore.Limit
var ps *pool.PoolStore
var bookingDuration, mocktime, startime int64
var useLocal bool
var bearer, secret string
var bc *apiclient.Bc

func init() {

	useLocal = true

	debug = false

	if debug {
		os.Setenv("DEBUG", "true") //for apiclient
		log.SetReportCaller(true)
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&logrus.TextFormatter{FullTimestamp: false, DisableColors: true})
		defer log.SetOutput(os.Stdout)

	} else {
		os.Setenv("DEBUG", "false")
		log.SetLevel(log.WarnLevel)
		var ignore bytes.Buffer
		logignore := bufio.NewWriter(&ignore)
		log.SetOutput(logignore)
	}

}

func TestMain(m *testing.M) {

	var lt lit.Token
	var cfg *apiclient.TransportConfig
	var loginBearer string

	iat := time.Now().Unix() - 1
	nbf := iat
	exp := nbf + 30

	if useLocal {

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		secret = "somesecret"

		ps = pool.NewPoolStore().
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
		loginBearer, err = lit.Signed(lt, secret)
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
		loginBearer, err = lit.Signed(lt, secret)
		if err != nil {
			panic(err)
		}

		cfg = apiclient.DefaultTransportConfig().WithSchemes([]string{"https"})

	}

	loginAuth := httptransport.APIKeyAuth("Authorization", "header", loginBearer)

	bc := apiclient.NewHTTPClientWithConfig(nil, cfg)

	timeout := 10 * time.Second

	params := login.NewLoginParams().WithTimeout(timeout)
	resp, err := bc.Login.Login(params, loginAuth)
	if err != nil {
		panic(err)
	}

	bearer = *resp.GetPayload().Token

	exitVal := m.Run()

	os.Exit(exitVal)
}

func TestExample(t *testing.T) {
	exp := int64(1613256113)
	m0 := Example(exp)

	_, err := yaml.Marshal(m0)

	assert.NoError(t, err)

	content, err := ioutil.ReadFile("testdata/example.yaml")
	assert.NoError(t, err)

	m1 := &Manifest{}
	err = yaml.Unmarshal(content, m1)
	assert.NoError(t, err)

	assert.Equal(t, m0, m1)

}

func TestBearer(t *testing.T) {
	// admin
	token, err := jwt.ParseWithClaims(bearer, &lit.Token{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)

	claims, ok := token.Claims.(*lit.Token)

	assert.True(t, ok)
	assert.True(t, token.Valid)

	assert.Equal(t, []string{"everyone"}, claims.Groups)
	assert.Equal(t, []string{"booking:admin"}, claims.Scopes)
	assert.True(t, claims.ExpiresAt > time.Now().Unix()+30)

	_, err = uuid.Parse(claims.Subject)
	assert.NoError(t, err)

}
