package manifest

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

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
	"gopkg.in/yaml.v2"
)

var debug bool
var l *bookingstore.Limit
var ps *pool.PoolStore
var host, audience, localSecret, remoteSecret string
var bookingDuration, mocktime, startime int64
var useLocal bool

func init() {

	useLocal = false

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	localSecret = "somesecret"
	secret, err := ioutil.ReadFile("../../secret/book.practable.io.secret")
	if err != nil {
		panic(err)
	}
	remoteSecret = strings.TrimSuffix(string(secret), "\n")

	bookingDuration = int64(180)

	mocktime = time.Now().Unix()
	startime = mocktime

	ps = pool.NewPoolStore().
		WithSecret(localSecret).
		WithBookingTokenDuration(bookingDuration).
		WithNow(func() int64 { return func(now *int64) int64 { return *now }(&mocktime) })

	l = bookingstore.New(ctx).WithFlush(time.Minute).WithMax(2).WithProvisionalPeriod(5 * time.Second)

	port, err := freeport.GetFreePort()
	if err != nil {
		panic(err)
	}

	host = "[::]:" + strconv.Itoa(port)

	audience = "http://" + host

	go booking.API(ctx, port, audience, localSecret, ps, l)

	time.Sleep(time.Second)

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

func TestBc(t *testing.T) {

	iat := time.Now().Unix() - 1
	nbf := iat
	exp := nbf + 30

	var lt lit.Token
	var bearer string
	var err error
	if useLocal {
		lt = lit.NewToken(audience, []string{"everyone"}, []string{}, []string{"login:admin"}, iat, nbf, exp)
		bearer, err = lit.Signed(lt, localSecret)
	} else {
		lt = lit.NewToken("https://book.practable.io", []string{"everyone"}, []string{}, []string{"login:admin"}, iat, nbf, exp)
		bearer, err = lit.Signed(lt, remoteSecret)
	}

	assert.NoError(t, err)

	loginAuth := httptransport.APIKeyAuth("Authorization", "header", bearer)

	var cfg *apiclient.TransportConfig

	if useLocal {
		cfg = apiclient.DefaultTransportConfig().WithHost(host).WithSchemes([]string{"http"})
	} else {
		cfg = apiclient.DefaultTransportConfig().WithSchemes([]string{"https"})
	}

	bc := apiclient.NewHTTPClientWithConfig(nil, cfg)

	timeout := 10 * time.Second
	params := login.NewLoginParams().WithTimeout(timeout)

	resp, err := bc.Login.Login(params, loginAuth)

	assert.NoError(t, err)

	fmt.Println(resp)

	//rehydrate, get the bookingToken ...

}
