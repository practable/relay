package status

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/phayes/freeport"
	"github.com/practable/relay/internal/permission"
	"github.com/practable/relay/internal/relay"
	"github.com/practable/relay/pkg/client"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {

	log.SetLevel(log.WarnLevel)

}

func makeTestToken(audience, secret string, ttl int64) (string, error) {

	var claims permission.Token

	start := jwt.NewNumericDate(time.Now().Add(-time.Second))
	afterTTL := jwt.NewNumericDate(time.Now().Add(time.Duration(ttl) * time.Second))
	claims.IssuedAt = start
	claims.NotBefore = start
	claims.ExpiresAt = afterTTL
	claims.Audience = jwt.ClaimStrings{audience}
	claims.Topic = "123"
	claims.ConnectionType = "session"
	claims.Scopes = []string{"read", "write"}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign and get the complete encoded token as a string using the secret
	return token.SignedString([]byte(secret))
}

func TestUnmarshal(t *testing.T) {

	m := []byte(`{"topic":"stats","canRead":true,"canWrite":true,"connected":"2023-03-10T14:04:45.294633437Z","expiresAt":"2023-03-10T14:04:45.294633437Z","remoteAddr":"internal","userAgent":"crossbar","stats":{"tx":{"last":"Never","size":0,"fps":0},"rx":{"last":"Never","size":0,"fps":0}}}`)

	ma := []byte(`[{"topic":"stats","canRead":true,"canWrite":true,"connected":"2023-03-10T14:04:45.294633437Z","expiresAt":"2023-03-10T14:04:45.294633437Z","remoteAddr":"internal","userAgent":"crossbar","stats":{"tx":{"last":"Never","size":0,"fps":0},"rx":{"last":"Never","size":0,"fps":0}}},{"topic":"stats","canRead":true,"canWrite":false,"connected":"2023-03-10T14:04:48.102847403Z","expiresAt":"2023-03-10T14:04:48.102847403Z","remoteAddr":"","userAgent":"Go-http-client/1.1","stats":{"tx":{"last":"Never","size":0,"fps":0},"rx":{"last":"Never","size":0,"fps":0}}},{"topic":"123","canRead":true,"canWrite":true,"connected":"2023-03-10T14:04:46.294717207Z","expiresAt":"2023-03-10T14:04:46.294717207Z","remoteAddr":"","userAgent":"Go-http-client/1.1","stats":{"tx":{"last":"2.90373838s","size":5,"fps":20.173318460352565},"rx":{"last":"2.903832419s","size":5,"fps":20.192179268105992}}},{"topic":"123","canRead":true,"canWrite":true,"connected":"2023-03-10T14:04:46.29484872Z","expiresAt":"2023-03-10T14:04:46.29484872Z","remoteAddr":"","userAgent":"Go-http-client/1.1","stats":{"tx":{"last":"2.903868479s","size":5,"fps":20.225188461845494},"rx":{"last":"2.903726512s","size":4,"fps":10.097985601322723}}}]`)

	mb := []byte(`{"topic":"stats","canRead":true,"canWrite":true,"connected":"2023-03-10T14:04:45.294633437Z","expiresAt":"2023-03-10T14:04:45.294633437Z","remoteAddr":"internal","userAgent":"crossbar","stats":{"tx":{"last":3600000000000,"size":0,"fps":0},"rx":{"last":100000000,"size":0,"fps":0}}}`)

	assert.True(t, json.Valid(m))
	assert.True(t, json.Valid(ma))
	assert.True(t, json.Valid(mb))

	var report Report

	err := json.Unmarshal(m, &report)

	assert.NoError(t, err)

	var reports []Report

	err = json.Unmarshal(ma, &reports)

	assert.NoError(t, err)
	err = json.Unmarshal(mb, &report)

	assert.NoError(t, err)

	assert.Equal(t, time.Hour, report.Stats.Tx.Last)
}

func TestStatus(t *testing.T) {

	// Setup logging
	debug := false

	if debug {
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true, DisableColors: true})
		defer log.SetOutput(os.Stdout)

	} else {
		var ignore bytes.Buffer
		logignore := bufio.NewWriter(&ignore)
		log.SetOutput(logignore)
	}

	// Setup relay on local (free) port
	closed := make(chan struct{})
	var wg sync.WaitGroup

	ports, err := freeport.GetFreePorts(2)
	assert.NoError(t, err)

	relayPort := ports[0]
	accessPort := ports[1]

	audience := "http://[::]:" + strconv.Itoa(accessPort)
	target := "ws://127.0.0.1:" + strconv.Itoa(relayPort)

	secret := "testsecret"

	wg.Add(1)

	go func() {
		time.Sleep(2 * time.Second)
		config := relay.Config{
			AccessPort:       accessPort,
			RelayPort:        relayPort,
			Audience:         audience,
			Secret:           secret,
			Target:           target,
			AllowNoBookingID: true,
			PruneEvery:       time.Duration(time.Minute),
			StatsEvery:       time.Duration(time.Second),
		}
		go relay.Relay(closed, &wg, config)
	}()

	// we sleep before starting the relay to help avoid issues with multiple
	// handlers registering with net/http when running all tests

	// Sign and get the complete encoded token as a string using the secret
	token, err := makeTestToken(audience, secret, 30)

	assert.NoError(t, err)

	// now clients connect using their uris...

	var timeout = 100 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())

	to := audience + "/session/123"

	// wait until relay has been up for about one second
	time.Sleep(3 * time.Second)

	c0 := client.New()
	go c0.Connect(ctx, to, token)

	c1 := client.New()
	go c1.Connect(ctx, to, token)
	// now wait until both clients have connected
	// one will connect before the other, so it's not
	// possible to guarantee both get this first message
	// and that is normal behaviour for a non-caching
	// relay....
	data := []byte("hello")
	c0.Send <- client.Message{Content: data, Type: client.TextMessage}
	c1.Send <- client.Message{Content: data, Type: client.TextMessage}

	time.Sleep(timeout) // send can come online before receive

	// now send a message we care about
	data0 := []byte("ping")
	c0.Send <- client.Message{Content: data0, Type: client.TextMessage}
	data1 := []byte("pong")
	c1.Send <- client.Message{Content: data1, Type: client.TextMessage}

	gotPing := false
	gotPong := false

	for i := 0; i < 20; i++ {
		select {
		case msg := <-c1.Receive:
			if debug {
				t.Log(string(msg.Content))
			}
			if bytes.Equal(msg.Content, data0) {
				gotPing = true
			}
			// sometimes the messages combine into "helloping"
			// due to the way framing is estimated in relay
			if bytes.Equal(msg.Content, append(data, data0...)) {
				gotPing = true
			}
		case msg := <-c0.Receive:
			if debug {
				t.Log(string(msg.Content))
			}
			if bytes.Equal(msg.Content, data1) {
				gotPong = true
				if gotPing {
					break
				}
			}
			if bytes.Equal(msg.Content, append(data, data1...)) {
				gotPing = true
				if gotPong {
					break
				}
			}
		case <-time.After(timeout):
			continue
		}
	}

	if !gotPing || !gotPong {
		t.Error("did not get both messages")
	}

	// Now that we have had some messaging, check the stats
	var claims permission.Token

	start := jwt.NewNumericDate(time.Now().Add(-time.Second))
	afterTTL := jwt.NewNumericDate(time.Now().Add(time.Duration(60) * time.Second))
	claims.IssuedAt = start
	claims.NotBefore = start
	claims.ExpiresAt = afterTTL
	claims.Audience = jwt.ClaimStrings{audience}
	claims.Topic = "stats"
	claims.ConnectionType = "session"
	claims.Scopes = []string{"read"}

	rawtoken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign and get the complete encoded token as a string using the secret
	token, err = rawtoken.SignedString([]byte(secret))

	assert.NoError(t, err)

	s := New()
	to = audience + "/session/stats"
	go s.Connect(ctx, to, token)

	select {
	case <-time.After(2 * time.Second):
		t.Error("did not receive status report in time")
	case report := <-s.Status:

		if debug {
			fmt.Printf("status: %+v\n", report)
		}

		// check we got a real set of reports, by checking for the existence
		// of the optics that currently apply
		ta := make(map[string]bool)

		for _, r := range report {
			ta[r.Topic] = true
		}

		te := map[string]bool{
			"123":   true,
			"stats": true,
		}

		assert.Equal(t, te, ta)

	}

	cancel()
	// Shutdown the Relay and check no messages are being sent
	close(closed)
	wg.Wait()

}

func TestMocking(t *testing.T) {

	s := New()

	tr := []Report{Report{Topic: "test00"}}
	go func() {
		s.Status <- tr
	}()

	select {
	case <-time.After(time.Second):
		t.Error("did not receive report")

	case ar := <-s.Status:

		assert.Equal(t, tr, ar)
	}

}
