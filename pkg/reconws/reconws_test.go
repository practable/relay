package reconws

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/websocket"
	"github.com/jpillora/backoff"
	"github.com/phayes/freeport"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/timdrysdale/relay/pkg/permission"
	"github.com/timdrysdale/relay/pkg/relay"
)

func init() {

	log.SetLevel(log.WarnLevel)

}

func makeTestToken(audience, secret string, ttl int64) (string, error) {

	var claims permission.Token

	claims.IssuedAt = time.Now().Unix() - 1
	claims.NotBefore = time.Now().Unix() - 1
	claims.ExpiresAt = time.Now().Unix() + ttl
	claims.Audience = audience
	claims.Topic = "123"
	claims.ConnectionType = "session"
	claims.Scopes = []string{"read", "write"}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign and get the complete encoded token as a string using the secret
	return token.SignedString([]byte(secret))
}

func TestBackoff(t *testing.T) {

	b := &backoff.Backoff{
		Min:    time.Second,
		Max:    30 * time.Second,
		Factor: 2,
		Jitter: false,
	}

	lowerBound := []float64{0.5, 1.5, 3.5, 7.5, 15.5, 29.5, 29.5, 29.5}
	upperBound := []float64{1.5, 2.5, 4.5, 8.5, 16.5, 30.5, 30.5, 30.5}

	for i := 0; i < len(lowerBound); i++ {

		actual := big.NewFloat(b.Duration().Seconds())

		if actual.Cmp(big.NewFloat(upperBound[i])) > 0 {
			t.Errorf("retry timing was incorrect, iteration %d, elapsed %f, wanted <%f\n", i, actual, upperBound[i])
		}
		if actual.Cmp(big.NewFloat(lowerBound[i])) < 0 {
			t.Errorf("retry timing was incorrect, iteration %d, elapsed %f, wanted >%f\n", i, actual, lowerBound[i])
		}

	}

}

func TestReconnectAuth(t *testing.T) {

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

	startTime := time.Now().Unix()
	go func() {
		time.Sleep(2 * time.Second)
		go relay.Relay(closed, &wg, accessPort, relayPort, audience, secret, target)
	}()

	// We can't start, stop and restart the relay.Relay without causing mux issues due to net/http
	// It panics on registration of multiple handlers
	// so start with it not running, then after some time,
	// and attempts have been made to connect - start relay
	// and see if the ReconnectAuth clients will connect.

	// Sign and get the complete encoded token as a string using the secret
	bearer, err := makeTestToken(audience, secret, 30)

	assert.NoError(t, err)

	// now clients connect using their uris...

	var timeout = 100 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())

	s0 := New()
	go s0.ReconnectAuth(ctx, audience+"/session/123", bearer)

	s1 := New()
	go s1.ReconnectAuth(ctx, audience+"/session/123", bearer)

	time.Sleep(timeout)

	data := []byte("prestart-ping-no-chance")

	select {
	case s0.Out <- WsMessage{Data: data, Type: websocket.TextMessage}:
		t.Fatal("s0 sent messsage to dead relay")
		select {
		case <-s1.In:
			t.Fatal("no message expected")
		case <-time.After(timeout):
		}
	case <-time.After(timeout):
	}

	data = []byte("prestart-pong-no-chance")

	select { //send may or may not happen
	case s1.Out <- WsMessage{Data: data, Type: websocket.TextMessage}:
		t.Fatal("s1 sent messsage to dead relay")
		select {
		case <-s0.In:
			t.Fatal("no message expected")
		case <-time.After(timeout):
		}
	case <-time.After(timeout):
	}

	// check we finished test before relay started
	assert.True(t, startTime+2 > time.Now().Unix())

	// now wait until both clients have connected
	// one will connect before the other, so it's not
	// possible to guarantee both get this first message
	// and that is normal behaviour for a non-caching
	// relay....
	data = []byte("hello")
	s0.Out <- WsMessage{Data: data, Type: websocket.TextMessage}
	s1.Out <- WsMessage{Data: data, Type: websocket.TextMessage}

	time.Sleep(timeout) // send can come online before receive

	// now send a message we care about
	data0 := []byte("ping")
	s0.Out <- WsMessage{Data: data0, Type: websocket.TextMessage}
	data1 := []byte("pong")
	s1.Out <- WsMessage{Data: data1, Type: websocket.TextMessage}

	gotPing := false
	gotPong := false

	for i := 0; i < 20; i++ {
		select {
		case msg := <-s1.In:
			if debug {
				t.Log(string(msg.Data))
			}
			if bytes.Equal(msg.Data, data0) {
				gotPing = true
			}
			// sometimes the messages combine into "helloping"
			// due to the way framing is etsimated in relay
			if bytes.Equal(msg.Data, append(data, data0...)) {
				gotPing = true
			}
		case msg := <-s0.In:
			if debug {
				t.Log(string(msg.Data))
			}
			if bytes.Equal(msg.Data, data1) {
				gotPong = true
				if gotPing {
					break
				}
			}
			if bytes.Equal(msg.Data, append(data, data1...)) {
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

func TestWsEcho(t *testing.T) {

	r := New()

	// Create test server with the echo handler.
	s := httptest.NewServer(http.HandlerFunc(echo))
	defer s.Close()

	// Convert http://127.0.0.1 to ws://127.0.0.
	u := "ws" + strings.TrimPrefix(s.URL, "http")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	go r.Reconnect(ctx, u)

	payload := []byte("Hello")
	mtype := int(websocket.TextMessage)

	r.Out <- WsMessage{Data: payload, Type: mtype}

	reply := <-r.In

	if !bytes.Equal(reply.Data, payload) {
		t.Errorf("Got unexpected response: %s, wanted %s\n", reply.Data, payload)
	}

	time.Sleep(2 * time.Second)

}

func TestRetryTiming(t *testing.T) {

	suppressLog()
	defer displayLog()

	r := New()

	c := make(chan int)

	// Create test server with the echo handler.
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deny(w, r, c)
	}))
	defer s.Close()

	// Convert http://127.0.0.1 to ws://127.0.0.
	url := "ws" + strings.TrimPrefix(s.URL, "http")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Reconnect(ctx, url)

	// first failed connection should be immediate
	// backoff with jitter means we quite can't be sure what the timings are
	lowerBound := []float64{0.0, 0.9, 1.9, 3.9, 7.9, 9.9, 9.9, 9.9}
	upperBound := []float64{1.0, 1.1, 2.1, 4.1, 8.1, 10.1, 10.1, 10.1}

	iterations := len(lowerBound)

	if testing.Short() {
		fmt.Println("Reducing length of test in short mode")
		iterations = 3
	}

	if testing.Verbose() {
		fmt.Println("lower < actual < upper ok?")
	}

	for i := 0; i < iterations; i++ {

		start := time.Now()

		<-c // wait for deny handler to return a value (note: bad handshake due to use of deny handler)

		actual := big.NewFloat(time.Since(start).Seconds())
		ok := true

		if actual.Cmp(big.NewFloat(upperBound[i])) > 0 {
			t.Errorf("retry timing was incorrect, iteration %d, elapsed %f, wanted <%f\n", i, actual, upperBound[i])
			ok = false
		}
		if actual.Cmp(big.NewFloat(lowerBound[i])) < 0 {
			t.Errorf("retry timing was incorrect, iteration %d, elapsed %f, wanted >%f\n", i, actual, lowerBound[i])
			ok = false
		}

		if testing.Verbose() {
			fmt.Printf("%0.2f < %0.2f < %0.2f %s\n", lowerBound[i], actual, upperBound[i], okString(ok))
		}
	}

}

func TestReconnectAfterDisconnect(t *testing.T) {

	r := New()

	c := make(chan int)

	n := 0

	// Create test server with the echo handler.
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connectAfterTrying(w, r, &n, 2, c)
	}))
	defer s.Close()

	// Convert http://127.0.0.1 to ws://127.0.0.
	url := "ws" + strings.TrimPrefix(s.URL, "http")

	ctx, cancel := context.WithCancel(context.Background()) //, time.Second)
	go r.Reconnect(ctx, url)

	// first failed connection should be immediate
	// should connect on third try
	// then next attempt after that should fail immediately
	// backoff with jitter means we quite can't be sure what the timings are
	// fail immediately, wait retry and fail, wait retry and connect, fail immediately, wait retry and fail
	lowerBound := []float64{0.0, 0.9, 1.9, 0.0, 0.9, 1.9}
	upperBound := []float64{0.1, 1.1, 2.1, 0.1, 1.1, 2.1}

	iterations := len(lowerBound)

	if testing.Short() {
		fmt.Println("Reducing length of test in short mode")
		iterations = 6
	}

	if testing.Verbose() {
		fmt.Println("lower < actual < upper ok?")
	}

	for i := 0; i < iterations; i++ {

		start := time.Now()

		<-c // wait for deny handler to return a value (note: bad handshake due to use of deny handler)

		actual := big.NewFloat(time.Since(start).Seconds())
		ok := true
		if actual.Cmp(big.NewFloat(upperBound[i])) > 0 {
			t.Errorf("retry timing was incorrect, iteration %d, elapsed %f, wanted <%f\n", i, actual, upperBound[i])
			ok = false
		}
		if actual.Cmp(big.NewFloat(lowerBound[i])) < 0 {
			t.Errorf("retry timing was incorrect, iteration %d, elapsed %f, wanted >%f\n", i, actual, lowerBound[i])
			ok = false
		}
		if testing.Verbose() {
			fmt.Printf("%0.2f < %0.2f < %0.2f %s\n", lowerBound[i], actual, upperBound[i], okString(ok))
		}
	}
	cancel()
}

var upgrader = websocket.Upgrader{}

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		err = c.WriteMessage(mt, message)
		if err != nil {
			break
		}
	}
}

func deny(w http.ResponseWriter, r *http.Request, c chan int) {
	c <- 0
}

func connectAfterTrying(w http.ResponseWriter, r *http.Request, n *int, connectAt int, c chan int) {

	defer func() { *n++ }()

	c <- *n

	if *n == connectAt {

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		defer conn.Close()

		// immediately close
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	}
}

func suppressLog() {
	var ignore bytes.Buffer
	logignore := bufio.NewWriter(&ignore)
	log.SetOutput(logignore)
}

func displayLog() {
	log.SetOutput(os.Stdout)
}

func okString(ok bool) string {
	if ok {
		return "  ok"
	}
	return "  FAILED"
}
