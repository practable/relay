package reconws

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jpillora/backoff"
	log "github.com/sirupsen/logrus"
	crossbar "github.com/timdrysdale/relay/pkg/crossbar/cmd"
)

var testAuthToken string = "some.test.token"

func init() {

	log.SetLevel(log.PanicLevel)

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

func TestAuthBad(t *testing.T) {

	r := New()

	// Create test server with the echo handler.
	s := httptest.NewServer(http.HandlerFunc(auth))
	defer s.Close()

	// Convert http://127.0.0.1 to ws://127.0.0.
	u := "ws" + strings.TrimPrefix(s.URL, "http")
	token := "not.the.right.token"

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	go r.ReconnectAuth(ctx, u, token)

	payload := []byte("Hello")
	mtype := int(websocket.TextMessage)

	select {
	case <-time.After(1000 * time.Millisecond):
		// all ok
	case r.Out <- WsMessage{Data: payload, Type: mtype}:
		t.Errorf("Did not expect to be able to send a message when not authorised")
	}

	select {
	case <-time.After(50 * time.Millisecond):
		// all ok
	case _ = <-r.In:
		t.Errorf("Did not expect get a reply to message we did not send")
	}

	time.Sleep(2 * time.Second)

}

func TestAuth(t *testing.T) {

	r := New()

	// Create test server with the echo handler.
	s := httptest.NewServer(http.HandlerFunc(auth))
	defer s.Close()

	// Convert http://127.0.0.1 to ws://127.0.0.
	u := "ws" + strings.TrimPrefix(s.URL, "http")
	token := testAuthToken

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	go r.ReconnectAuth(ctx, u, token)

	payload := []byte("Hello")
	mtype := int(websocket.TextMessage)

	r.Out <- WsMessage{Data: payload, Type: mtype}

	reply := <-r.In

	if bytes.Compare(reply.Data, payload) != 0 {
		t.Errorf("Got unexpected response: %s, wanted %s\n", reply.Data, payload)
	}

	time.Sleep(2 * time.Second)

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

	if bytes.Compare(reply.Data, payload) != 0 {
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

func auth(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()

	mt, message, err := c.ReadMessage()
	if err != nil {
		return
	}

	reply := crossbar.AuthMessage{
		Authorised: false,
		Token:      testAuthToken,
		Reason:     "Denied", //not an official message ...
	}

	if string(message) == testAuthToken {
		reply = crossbar.AuthMessage{
			Authorised: true,
			Reason:     "ok",
		}
	}

	message, err = json.Marshal(&reply)

	err = c.WriteMessage(mt, message)
	if err != nil {

		return
	}

	//now echo
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

	defer func() { *n += 1 }()

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
	} else {
		return "  FAILED"
	}

}
