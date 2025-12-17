package client

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/phayes/freeport"
	"github.com/practable/relay/internal/permission"
	"github.com/practable/relay/internal/relay"
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

func TestClientConnect(t *testing.T) {

	// Setup logging
	debug := false

	if debug {
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true, DisableColors: true})
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

	c0 := New()
	go c0.Connect(ctx, to, token)

	c1 := New()
	go c1.Connect(ctx, to, token)
	// now wait until both clients have connected
	// one will connect before the other, so it's not
	// possible to guarantee both get this first message
	// and that is normal behaviour for a non-caching
	// relay....
	data := []byte("hello")
	c0.Send <- Message{Content: data, Type: TextMessage}
	c1.Send <- Message{Content: data, Type: TextMessage}

	time.Sleep(timeout) // send can come online before receive

	// now send a message we care about
	data0 := []byte("ping")
	c0.Send <- Message{Content: data0, Type: TextMessage}
	data1 := []byte("pong")
	c1.Send <- Message{Content: data1, Type: TextMessage}

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

	cancel()
	// Shutdown the Relay and check no messages are being sent
	close(closed)
	wg.Wait()

}
