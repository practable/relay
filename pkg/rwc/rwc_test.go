package rwc

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"
	"github.com/phayes/freeport"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/timdrysdale/relay/pkg/access"
	"github.com/timdrysdale/relay/pkg/agg"
	"github.com/timdrysdale/relay/pkg/hub"
	"github.com/timdrysdale/relay/pkg/permission"
	"github.com/timdrysdale/relay/pkg/reconws"
	"github.com/timdrysdale/relay/pkg/relay"
)

var testAuthToken string = "some.test.token"

func init() {

	log.SetLevel(log.ErrorLevel)

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

func TestInstantiateHub(t *testing.T) {

	mh := agg.New()

	h := New(mh)

	if reflect.TypeOf(h.Broadcast) != reflect.TypeOf(make(chan hub.Message)) {
		t.Error("Hub.Broadcast channel of wrong type")
	}
	if reflect.TypeOf(h.Clients) != reflect.TypeOf(make(map[string]*Client)) {
		t.Error("Hub.Clients map of wrong type")
	}
	if reflect.TypeOf(h.Add) != reflect.TypeOf(make(chan Rule)) {
		t.Error("Hub.Add channel of wrong type")
	}

	if reflect.TypeOf(h.Delete) != reflect.TypeOf(make(chan string)) {
		t.Errorf("Hub.Delete channel of wrong type wanted/got %v %v", reflect.TypeOf(""), reflect.TypeOf(h.Delete))
	}

	if reflect.TypeOf(h.Rules) != reflect.TypeOf(make(map[string]Rule)) {
		t.Error("Hub.Broadcast channel of wrong type")
	}

}

func TestAddRuleAuth(t *testing.T) {

	mh := agg.New()
	h := New(mh)

	closed := make(chan struct{})
	defer close(closed)

	go h.Run(closed)

	// Create test server with the echo handler.
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		echo(w, r)
	}))
	defer s.Close()

	id := "rule0"
	stream := "stream/large"
	destination := "ws" + strings.TrimPrefix(s.URL, "http") //s.URL //"ws://localhost:8081"
	token := "some.test.token"
	r := &Rule{Id: id,
		Stream:      stream,
		Destination: destination,
		Token:       token,
	}

	h.Add <- *r

	time.Sleep(time.Millisecond)

	if _, ok := h.Rules[id]; !ok {
		t.Error("Rule not registered in Rules")

	} else {
		assert.Equal(t, h.Rules[id].Destination, destination)
		assert.Equal(t, h.Rules[id].Stream, stream)
		assert.Equal(t, h.Rules[id].Token, token)
	}
}

func TestAddRule(t *testing.T) {

	mh := agg.New()
	h := New(mh)

	closed := make(chan struct{})
	defer close(closed)

	go h.Run(closed)

	// Create test server with the echo handler.
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		echo(w, r)
	}))
	defer s.Close()

	id := "rule0"
	stream := "stream/large"
	destination := "ws" + strings.TrimPrefix(s.URL, "http") //s.URL //"ws://localhost:8081"

	r := &Rule{Id: id,
		Stream:      stream,
		Destination: destination}

	h.Add <- *r

	time.Sleep(time.Millisecond)

	if _, ok := h.Rules[id]; !ok {
		t.Error("Rule not registered in Rules")

	} else {

		if h.Rules[id].Destination != destination {
			t.Errorf("Rule has incorrect destination wanted/got %v %v\n", destination, h.Rules[id].Destination)
		}
		if h.Rules[id].Stream != stream {
			t.Errorf("Rule has incorrect stream wanted/got %v %v\n", stream, h.Rules[id].Stream)
		}
	}
}

func TestCannotAddDeleteAllRule(t *testing.T) {

	mh := agg.New()
	h := New(mh)

	closed := make(chan struct{})
	defer close(closed)

	go h.Run(closed)

	// Create test server with the echo handler.
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		echo(w, r)
	}))
	defer s.Close()

	id := "deleteAll"
	stream := "stream/large"
	destination := "ws" + strings.TrimPrefix(s.URL, "http") //s.URL //"ws://localhost:8081"

	r := &Rule{Id: id,
		Stream:      stream,
		Destination: destination}

	h.Add <- *r

	time.Sleep(time.Millisecond)

	if _, ok := h.Rules[id]; ok {
		t.Error("Rule deleteAll incorrectly accepted into Rules")

	}
}

func TestAddRules(t *testing.T) {

	suppressLog()
	defer displayLog()

	closed := make(chan struct{})
	defer close(closed)

	mh := agg.New()
	go mh.Run(closed)

	h := New(mh)
	go h.Run(closed)

	// Create test server with the echo handler.
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		echo(w, r)
	}))
	defer s1.Close()

	// Create test server with the echo handler.
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		echo(w, r)
	}))
	defer s2.Close()

	id := "rule0"
	stream := "stream/large"
	destination := "ws" + strings.TrimPrefix(s1.URL, "http") //s1.URL //"ws://localhost:8081"

	r := &Rule{Id: id,
		Stream:      stream,
		Destination: destination}

	h.Add <- *r

	time.Sleep(time.Millisecond)

	id2 := "rule1"
	stream2 := "stream/medium"
	destination2 := "ws" + strings.TrimPrefix(s2.URL, "http") //s2.URL //"ws://localhost:8082"

	r2 := &Rule{Id: id2,
		Stream:      stream2,
		Destination: destination2}

	h.Add <- *r2

	time.Sleep(time.Millisecond)

	if _, ok := h.Rules[id]; !ok {
		t.Error("Rule not registered in Rules")

	} else {

		if h.Rules[id].Destination != destination {
			t.Errorf("Rule has incorrect destination wanted/got %v %v\n", destination, h.Rules[id].Destination)
			fmt.Printf("%v\n", h.Rules)
		}
		if h.Rules[id].Stream != stream {
			t.Errorf("Rule has incorrect stream wanted/got %v %v\n", stream, h.Rules[id].Stream)
			fmt.Printf("%v\n", h.Rules)
		}
	}
	if _, ok := h.Rules[id2]; !ok {
		t.Error("Rule not registered in Rules")

	} else {

		if h.Rules[id2].Destination != destination2 {
			t.Errorf("Rule has incorrect destination wanted/got %v %v\n", destination2, h.Rules[id2].Destination)
			fmt.Printf("%v\n", h.Rules)
		}
		if h.Rules[id2].Stream != stream2 {
			t.Errorf("Rule has incorrect stream wanted/got %v %v\n", stream2, h.Rules[id2].Stream)
		}
	}

	time.Sleep(500 * time.Millisecond)

}

func TestAddDupeRule(t *testing.T) {

	suppressLog()
	defer displayLog()

	closed := make(chan struct{})
	defer close(closed)

	mh := agg.New()
	go mh.Run(closed)

	h := New(mh)
	go h.Run(closed)

	// Create test server with the echo handler.
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		echo(w, r)
	}))
	defer s1.Close()

	// Create test server with the echo handler.
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		echo(w, r)
	}))
	defer s2.Close()

	id := "rule0"
	stream := "stream/large"
	destination := "ws" + strings.TrimPrefix(s1.URL, "http") //s.URL //"ws://localhost:8082"

	r := &Rule{Id: id,
		Stream:      stream,
		Destination: destination}

	h.Add <- *r

	time.Sleep(time.Millisecond)

	id = "rule0"
	stream = "stream/medium"
	destination = "ws" + strings.TrimPrefix(s2.URL, "http") //://localhost:8082"

	r = &Rule{Id: id,
		Stream:      stream,
		Destination: destination}

	h.Add <- *r

	time.Sleep(time.Millisecond)

	if _, ok := h.Rules[id]; !ok {
		t.Error("Rule not registered in Rules")

	} else {

		if h.Rules[id].Destination != destination {
			t.Errorf("Rule has incorrect destination wanted/got %v %v\n", destination, h.Rules[id].Destination)
		}
		if h.Rules[id].Stream != stream {
			t.Errorf("Rule has incorrect stream wanted/got %v %v\n", stream, h.Rules[id].Stream)
		}
	}

}

func TestDeleteRule(t *testing.T) {
	suppressLog()
	defer displayLog()

	closed := make(chan struct{})
	defer close(closed)

	mh := agg.New()
	go mh.Run(closed)

	h := New(mh)
	go h.Run(closed)

	// Create test server with the echo handler.
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		echo(w, r)
	}))
	defer s1.Close()

	// Create test server with the echo handler.
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		echo(w, r)
	}))
	defer s2.Close()

	id := "rule0"
	stream := "stream/large"
	destination := "ws" + strings.TrimPrefix(s1.URL, "http") //s1.URL //"ws://localhost:8081"

	r := &Rule{Id: id,
		Stream:      stream,
		Destination: destination}

	h.Add <- *r

	time.Sleep(time.Millisecond)

	id2 := "rule1"
	stream2 := "stream/medium"
	destination2 := "ws" + strings.TrimPrefix(s2.URL, "http") //s2.URL //"ws://localhost:8082"

	r2 := &Rule{Id: id2,
		Stream:      stream2,
		Destination: destination2}

	h.Add <- *r2

	time.Sleep(time.Millisecond)

	if _, ok := h.Rules[id]; !ok {
		t.Error("Rule not registered in Rules")

	} else {

		if h.Rules[id].Destination != destination {
			t.Errorf("Rule has incorrect destination wanted/got %v %v\n", destination, h.Rules[id].Destination)
		}
		if h.Rules[id].Stream != stream {
			t.Errorf("Rule has incorrect stream wanted/got %v %v\n", stream, h.Rules[id].Stream)
		}
	}
	if _, ok := h.Rules[id2]; !ok {
		t.Error("Rule not registered in Rules")

	} else {

		if h.Rules[id2].Destination != destination2 {
			t.Errorf("Rule has incorrect destination wanted/got %v %v\n", destination2, h.Rules[id2].Destination)
		}
		if h.Rules[id2].Stream != stream2 {
			t.Errorf("Rule has incorrect stream wanted/got %v %v\n", stream2, h.Rules[id2].Stream)
		}
	}

	h.Delete <- id

	time.Sleep(time.Millisecond)

	if _, ok := h.Rules[id]; ok {
		t.Error("Deleted rule registered in Rules")
	}

	if _, ok := h.Rules[id2]; !ok {
		t.Error("Rule not registered in Rules")
	} else {
		if h.Rules[id2].Destination != destination2 {
			t.Errorf("Rule has incorrect destination wanted/got %v %v\n", destination2, h.Rules[id2].Destination)
		}
		if h.Rules[id2].Stream != stream2 {
			t.Errorf("Rule has incorrect stream wanted/got %v %v\n", stream2, h.Rules[id2].Stream)
		}
	}
}

func TestDeleteAllRule(t *testing.T) {
	suppressLog()
	defer displayLog()

	closed := make(chan struct{})
	defer close(closed)

	mh := agg.New()
	go mh.Run(closed)

	h := New(mh)
	go h.Run(closed)

	// Create test server with the echo handler.
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		echo(w, r)
	}))
	defer s1.Close()

	// Create test server with the echo handler.
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		echo(w, r)
	}))
	defer s2.Close()

	id := "rule0"
	stream := "stream/large"
	destination := "ws" + strings.TrimPrefix(s1.URL, "http") //s1.URL //"ws://localhost:8081"

	r := &Rule{Id: id,
		Stream:      stream,
		Destination: destination}

	h.Add <- *r

	time.Sleep(time.Millisecond)

	id2 := "rule1"
	stream2 := "stream/medium"
	destination2 := "ws" + strings.TrimPrefix(s2.URL, "http") //s2.URL //"ws://localhost:8082"

	r2 := &Rule{Id: id2,
		Stream:      stream2,
		Destination: destination2}

	h.Add <- *r2

	time.Sleep(time.Millisecond)

	if _, ok := h.Rules[id]; !ok {
		t.Error("Rule not registered in Rules")

	} else {

		if h.Rules[id].Destination != destination {
			t.Errorf("Rule has incorrect destination wanted/got %v %v\n", destination, h.Rules[id].Destination)
		}
		if h.Rules[id].Stream != stream {
			t.Errorf("Rule has incorrect stream wanted/got %v %v\n", stream, h.Rules[id].Stream)
		}
	}
	if _, ok := h.Rules[id2]; !ok {
		t.Error("Rule not registered in Rules")

	} else {

		if h.Rules[id2].Destination != destination2 {
			t.Errorf("Rule has incorrect destination wanted/got %v %v\n", destination2, h.Rules[id2].Destination)
		}
		if h.Rules[id2].Stream != stream2 {
			t.Errorf("Rule has incorrect stream wanted/got %v %v\n", stream2, h.Rules[id2].Stream)
		}
	}

	h.Delete <- "deleteAll"

	time.Sleep(time.Millisecond)

	if _, ok := h.Rules[id]; ok {
		t.Error("Deleted rule registered in Rules")
	}
	if _, ok := h.Rules[id2]; ok {
		t.Error("Deleted rule registered in Rules")
	}
}

func TestConnectAuth(t *testing.T) {

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

	go relay.Relay(closed, &wg, accessPort, relayPort, audience, secret, target, access.Options{})

	// allow relay to start up
	time.Sleep(time.Second)

	timeout := 100 * time.Millisecond

	// Sign and get the complete encoded token as a string using the secret
	bearer, err := makeTestToken(audience, secret, 30)
	assert.NoError(t, err)

	destination := audience + "/session/123"

	// connect a client and have them waiting for a message

	ctx, cancel := context.WithCancel(context.Background())

	s1 := reconws.New()
	go s1.ReconnectAuth(ctx, audience+"/session/123", bearer)

	time.Sleep(timeout)

	// Now do the rwc thing...

	mh := agg.New()
	go mh.Run(closed)

	time.Sleep(time.Millisecond)

	h := New(mh)
	go h.Run(closed)

	id := "rule0"
	stream := "medium"
	token := bearer

	r := &Rule{Id: id,
		Stream:      stream,
		Destination: destination,
		Token:       token,
	}

	h.Add <- *r

	time.Sleep(time.Millisecond)

	if _, ok := h.Rules[id]; !ok {
		t.Error("Rule not registered in Rules")
	}

	reply := make(chan hub.Message)

	c := &hub.Client{Hub: mh.Hub, Name: "a", Topic: stream, Send: reply}

	h.Messages.Register <- c

	time.Sleep(1 * time.Millisecond)

	payload := []byte("test message")

	mh.Broadcast <- hub.Message{Data: payload, Type: websocket.TextMessage, Sender: *c, Sent: time.Now()}

	// Now we need to see if we got it...

	select {
	case msg := <-s1.In:
		assert.Equal(t, payload, msg.Data)
	case <-time.After(timeout):
		t.Fatal("No message")
	}

	cancel()
	// Shutdown the Relay and check no messages are being sent
	close(closed)
	wg.Wait()

}

func TestWriteToFile(t *testing.T) {

	// Create test server with the echo handler.
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		shout(w, r)
	}))

	closed := make(chan struct{})
	defer close(closed)

	mh := agg.New()
	go mh.Run(closed)

	time.Sleep(time.Millisecond)

	h := New(mh)
	go h.Run(closed)

	id := "rule0"
	stream := "medium"
	destination := "ws" + strings.TrimPrefix(s.URL, "http")
	file := "test.bin"

	r := &Rule{Id: id,
		Stream:      stream,
		Destination: destination,
		File:        file}

	h.Add <- *r

	time.Sleep(time.Millisecond)

	if _, ok := h.Rules[id]; !ok {
		t.Error("Rule not registered in Rules")
	}

	reply := make(chan hub.Message)

	c := &hub.Client{Hub: mh.Hub, Name: "a", Topic: stream, Send: reply}

	h.Messages.Register <- c

	time.Sleep(1 * time.Millisecond)

	string0 := "test message"

	payload := []byte(string0)
	shoutedPayload := []byte("TEST MESSAGE")

	mh.Broadcast <- hub.Message{Data: payload, Type: websocket.TextMessage, Sender: *c, Sent: time.Now()}

	select {
	case msg := <-reply:
		if bytes.Compare(msg.Data, shoutedPayload) != 0 {
			t.Error("Got wrong message")
		}
	case <-time.After(10 * time.Millisecond):
		t.Error("timed out waiting for message")
	}

	string1 := "here we go again"
	payload = []byte(string1)
	shoutedPayload = []byte("HERE WE GO AGAIN")

	mh.Broadcast <- hub.Message{Data: payload, Type: websocket.TextMessage, Sender: *c, Sent: time.Now()}

	select {
	case msg := <-reply:
		if bytes.Compare(msg.Data, shoutedPayload) != 0 {
			t.Error("Got wrong message")
		}
	case <-time.After(10 * time.Millisecond):
		t.Error("timed out waiting for message")
	}

	time.Sleep(10 * time.Millisecond)

	s.Close()

	time.Sleep(10 * time.Millisecond)

	dat, err := ioutil.ReadFile(file)
	assert.NoError(t, err)
	assert.Equal(t, []byte(string0+string1), dat)

}

func TestSendMessageTopic(t *testing.T) {

	// Create test server with the echo handler.
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		shout(w, r)
	}))
	defer s.Close()

	closed := make(chan struct{})
	defer close(closed)

	mh := agg.New()
	go mh.Run(closed)

	time.Sleep(time.Millisecond)

	h := New(mh)
	go h.Run(closed)

	id := "rule0"
	stream := "medium"
	destination := "ws" + strings.TrimPrefix(s.URL, "http")

	r := &Rule{Id: id,
		Stream:      stream,
		Destination: destination}

	h.Add <- *r

	time.Sleep(time.Millisecond)

	if _, ok := h.Rules[id]; !ok {
		t.Error("Rule not registered in Rules")
	}

	reply := make(chan hub.Message)

	c := &hub.Client{Hub: mh.Hub, Name: "a", Topic: stream, Send: reply}

	h.Messages.Register <- c

	time.Sleep(1 * time.Millisecond)

	payload := []byte("test message")
	shoutedPayload := []byte("TEST MESSAGE")

	mh.Broadcast <- hub.Message{Data: payload, Type: websocket.TextMessage, Sender: *c, Sent: time.Now()}

	select {
	case msg := <-reply:
		if bytes.Compare(msg.Data, shoutedPayload) != 0 {
			t.Error("Got wrong message")
		}
	case <-time.After(10 * time.Millisecond):
		t.Error("timed out waiting for message")
	}
}

func TestSendMessageStream(t *testing.T) {
	// Check whether a stream's aggregated messages can make it all the way to
	// a websocket server
	// Gotchas:
	//   -- streams do not send replies to feeds, so check websocket message directly
	//      after it has arrived at the server
	//   -- need matching stream rule (for agg) AND destination rule (for rwc)
	//
	//  +---------------+         +----------+           +----------+       +----------+
	//  |               |         |          |  "test"   |          |       |          |
	//  |   c0          |  "test" |          |  "TEST"   |          |       |          |
	//  |   hub.Client  +--------->          +----------->          |       |          |
	//  |               |         |          |           |          | "test"|          |
	//  +---------------+         |          |           |          | "TEST"|          |
	//                            |    agg   |           |   rwc    +-------> websocket|
	//                            |          |           |          |       | server   |
	//  +---------------+         |          |           |          |       |          |
	//  |   c1          |  "TEST" |          |  register |          |       |          |
	//  |   hub.Client  +--------->          +<----------+          |       | report)  |
	//  |               |         |          |           |          |       |          |
	//  +---------------+         +---^------+           +---^------+       +-----+----+
	//                                |                      |                    |
	//                                |                      |                    |
	//                                |                      |                    |
	//                                |                      |                    |
	//                                +                      +                    v
	//                          add stream rule          add dest rule         "test"
	//                                                                         "TEST"
	//
	//                                               --diagram created with asciiflow.com
	//
	// Note we are also testing that a server without a rule does not somehow also get messages
	// This is null-test before we do the following test (changing destinations)
	closed := make(chan struct{})
	defer close(closed)

	// aggregating hub
	mh := agg.New()
	go mh.Run(closed)

	time.Sleep(time.Millisecond)

	// rwc hub
	h := New(mh)
	go h.Run(closed)

	// destination websocket reporting channels
	wsMsg0 := make(chan reconws.WsMessage)
	wsMsg1 := make(chan reconws.WsMessage)

	// destination websocket servers
	s0 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { report(w, r, wsMsg0) }))
	defer s0.Close()

	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { report(w, r, wsMsg1) }))
	defer s1.Close()

	//destination rule
	id := "rule0"
	stream := "stream/medium"
	destination0 := "ws" + strings.TrimPrefix(s0.URL, "http")

	r0 := &Rule{Id: id,
		Stream:      stream,
		Destination: destination0}

	//stream rule
	feeds := []string{"video0", "audio"}
	streamRule := &agg.Rule{Stream: stream, Feeds: feeds}

	// receivers

	rxCount0 := 0
	rxCount1 := 0
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		select {
		case <-wsMsg0:
			rxCount0++
		case <-time.After(time.Second):
			t.Error("Timeout on destination websocket server 0")
		}
		for msg := range wsMsg0 {
			if len(msg.Data) > 0 {
				rxCount0++
			}
		}
	}()
	go func() {
		defer wg.Done()
		select {
		case <-wsMsg1:
			rxCount1++
		case <-time.After(time.Second):
			t.Error("Timeout on destination websocket server 0")
		}
		for msg := range wsMsg1 {
			if len(msg.Data) > 0 {
				rxCount1++
			}
		}
	}()

	// set up the stream
	mh.Add <- *streamRule

	// set up the destination websocket client
	h.Add <- *r0
	time.Sleep(time.Millisecond)

	// check on rule being in place
	if _, ok := h.Rules[id]; !ok {
		t.Error("Rule not registered in Rules")
	}

	// add clients/feeds to supply the stream
	reply0 := make(chan hub.Message)
	reply1 := make(chan hub.Message)

	c0 := &hub.Client{Hub: mh.Hub, Name: "video0", Topic: feeds[0], Send: reply0}
	c1 := &hub.Client{Hub: mh.Hub, Name: "audio", Topic: feeds[1], Send: reply1}

	h.Messages.Register <- c0
	h.Messages.Register <- c1

	time.Sleep(1 * time.Millisecond)

	payload := []byte("test message")
	shoutedPayload := []byte("TEST MESSAGE")

	// clients send messages ...
	for i := 0; i < 5; i++ {
		log.Debug("Sending message", i)
		c0.Hub.Broadcast <- hub.Message{Data: payload, Type: websocket.TextMessage, Sender: *c0, Sent: time.Now()}
		c1.Hub.Broadcast <- hub.Message{Data: shoutedPayload, Type: websocket.TextMessage, Sender: *c1, Sent: time.Now()}
		time.Sleep(time.Millisecond)
	}

	if rxCount0 != 10 {
		t.Errorf("Destination0 did not receive correct number of messages; wanted %d, got %d\n", 10, rxCount0)
	}
	if rxCount1 != 0 {
		t.Errorf("Destination1 did not receive correct number of messages; wanted %d, got %d\n", 0, rxCount0)
	}
	close(wsMsg0)
	close(wsMsg1)
	wg.Wait()
}

func TestSendMessageToChangingDestination(t *testing.T) {

	closed := make(chan struct{})
	defer close(closed)

	// aggregating hub
	mh := agg.New()
	go mh.Run(closed)

	time.Sleep(time.Millisecond)

	// rwc hub
	h := New(mh)
	go h.Run(closed)

	// destination websocket reporting channels
	wsMsg0 := make(chan reconws.WsMessage)
	wsMsg1 := make(chan reconws.WsMessage)

	// destination websocket servers
	s0 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { report(w, r, wsMsg0) }))
	defer s0.Close()

	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { report(w, r, wsMsg1) }))
	defer s1.Close()

	//destination rules
	id := "rule0"
	stream := "stream/medium"
	destination0 := "ws" + strings.TrimPrefix(s0.URL, "http")
	destination1 := "ws" + strings.TrimPrefix(s1.URL, "http")

	//stream rule
	feeds := []string{"video0", "audio"}
	streamRule := &agg.Rule{Stream: stream, Feeds: feeds}

	r0 := &Rule{Id: id,
		Stream:      stream,
		Destination: destination0}

	r1 := &Rule{Id: id,
		Stream:      stream,
		Destination: destination1}

	// receivers

	rxCount0 := 0
	rxCount1 := 0

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		select {
		case msg := <-wsMsg0:
			if len(msg.Data) > 0 {
				rxCount0++
			}
		case <-time.After(time.Second):
			t.Error("Timeout on destination websocket server 0")
		}
		for msg := range wsMsg0 {
			if len(msg.Data) > 0 {
				rxCount0++
			}
		}
	}()
	go func() {
		defer wg.Done()
		select {
		case msg := <-wsMsg1:
			if len(msg.Data) > 0 {
				rxCount1++
			}
		case <-time.After(time.Second):
			t.Error("Timeout on destination websocket server 0")
		}
		for msg := range wsMsg1 {
			if len(msg.Data) > 0 {
				rxCount1++
			}
		}
	}()

	// set up the stream
	mh.Add <- *streamRule

	// set up the destination websocket client
	h.Add <- *r0
	time.Sleep(time.Millisecond)

	// check on rule being in place
	if _, ok := h.Rules[id]; !ok {
		t.Error("Rule not registered in Rules")
	}

	// add clients/feeds to supply the stream
	reply0 := make(chan hub.Message)
	reply1 := make(chan hub.Message)

	c0 := &hub.Client{Hub: mh.Hub, Name: "video0", Topic: feeds[0], Send: reply0}
	c1 := &hub.Client{Hub: mh.Hub, Name: "audio", Topic: feeds[1], Send: reply1}

	// drain reply channels
	wg.Add(2)
	go func() {
		defer wg.Done()
		for _ = range reply0 {
		}

	}()
	go func() {
		defer wg.Done()
		for _ = range reply1 {
		}

	}()

	h.Messages.Register <- c0
	h.Messages.Register <- c1

	time.Sleep(1 * time.Millisecond)

	payload := []byte("test message")
	shoutedPayload := []byte("TEST MESSAGE")

	// clients send messages ...
	for i := 0; i < 5; i++ {
		log.Debug("Sending message", i)
		c0.Hub.Broadcast <- hub.Message{Data: payload, Type: websocket.TextMessage, Sender: *c0, Sent: time.Now()}
		c1.Hub.Broadcast <- hub.Message{Data: shoutedPayload, Type: websocket.TextMessage, Sender: *c1, Sent: time.Now()}
		time.Sleep(time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	// change the destination websocket client
	h.Add <- *r1
	log.Debug(r1)
	time.Sleep(1 * time.Millisecond)
	// check on rule being in place
	if _, ok := h.Rules[id]; !ok {
		t.Error("Rule not registered in Rules")
	} else {
		if h.Rules[id].Destination != destination1 {
			t.Errorf("Updated rule has wrong destination")
		}
	}
	time.Sleep(100 * time.Millisecond)
	// clients send messages ...
	for i := 0; i < 10; i++ {
		log.Debug("Sending message", i)
		c0.Hub.Broadcast <- hub.Message{Data: payload, Type: websocket.TextMessage, Sender: *c0, Sent: time.Now()}
		c1.Hub.Broadcast <- hub.Message{Data: shoutedPayload, Type: websocket.TextMessage, Sender: *c1, Sent: time.Now()}
		time.Sleep(time.Millisecond)
	}

	if rxCount0 != 10 {
		t.Errorf("Destination0 did not receive correct number of messages; wanted %d, got %d\n", 10, rxCount0)
	}
	if rxCount1 != 20 {
		t.Errorf("Destination1 did not receive correct number of messages; wanted %d, got %d\n", 20, rxCount1)
	}
	time.Sleep(10 * time.Millisecond)

	close(wsMsg0)
	close(wsMsg1)
	close(reply0)
	close(reply1)
	wg.Wait()
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

func report(w http.ResponseWriter, r *http.Request, msgChan chan reconws.WsMessage) {
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
		msgChan <- reconws.WsMessage{Data: message, Type: mt}
	}
}

func shout(w http.ResponseWriter, r *http.Request) {
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
		message = []byte(strings.ToUpper(string(message)))
		err = c.WriteMessage(mt, message)
		if err != nil {
			break
		}
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
