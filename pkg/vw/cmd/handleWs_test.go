package cmd

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/relay/pkg/agg"
	"github.com/timdrysdale/relay/pkg/hub"
	"github.com/timdrysdale/relay/pkg/reconws"
)

func init() {
	log.SetLevel(log.ErrorLevel)
}

func TestHandleWsSendMessageViaClient(t *testing.T) {

	closed := make(chan struct{})

	h := agg.New()
	go h.Run(closed)

	ctx := &hub.Client{Hub: h.Hub, Name: "tx", Topic: "/greetings", Send: make(chan hub.Message), Stats: hub.NewClientStats()}
	crx := &hub.Client{Hub: h.Hub, Name: "rx", Topic: "/greetings", Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	h.Register <- crx

	greeting := []byte("hello")

	m := &hub.Message{Sender: *ctx, Sent: time.Now(), Data: greeting, Type: websocket.TextMessage}
	h.Broadcast <- *m

	time.Sleep(time.Millisecond)

	select {
	case <-time.After(10 * time.Millisecond):
	case msg, ok := <-crx.Send:
		if ok {
			if !bytes.Equal(msg.Data, greeting) {
				t.Errorf("Greeting content unexpected; got/wanted %v/%v\n", string(msg.Data), string(greeting))
			}
		}
	}
}

func TestHandleWsSendMessageViaWs(t *testing.T) {
	// This was confusing when I came back to it, so here's a diagram:
	//
	//           +----------+       +----------+        +---------+             +----------+
	//           |    r     |       |          |        |         |             |          |
	//r.Out+---->+ (reconws |       | wsHandler|        |  Agg    | crx.Send    |  crx     |
	//           |  test    +------>+  (under  +------->+         +-------------> (hub     |
	// r.In<-----+  client) |       |   test)  |        |         |             |  test    |
	//           |          |       |          |        |         |             |  client) |
	//           +----------+       +----------+        +---------+             +----------+
	//
	// <-------- TEST HARNESS--><--ITEM UNDER TEST--><---TEST HARNESS---------------------->
	//
	//                                                 --diagram created using asciiflow.com
	//
	// Explanation:
	// What's under test is the wsHandler's ability to direct messages from an incoming websocket
	// connection to the appropriate topic in the agg (aggregating) hub.
	//
	// Test harness comprises a reconws client, to send the message, and an agg and a
	// standard hub.Client to receive the message
	//
	// The message is fed into the channel r.Out of the reconws test client, which sends it over
	// websocket to ws://locahost:<testport>/ws/greetings
	// crx, the hub test client, has registered to topic /greetings
	// so the wsHandler has to strip the leading /ws for this test to pass
	// obviously this is more of an integration test, because we are using agg and a hub.Client
	// which only really occurred to me after writing this more complete test.
	// Consequently, we also take the opportunity to check the reverse passage of messages and
	// have crx reply to the websocket client

	// Test harness, receiving side (agg, and hub.Client)
	app := App{Hub: agg.New(), Closed: make(chan struct{})}
	defer close(app.Closed)
	go app.Hub.Run(app.Closed)

	time.Sleep(2 * time.Millisecond)

	crx := &hub.Client{Hub: app.Hub.Hub, Name: "rx", Topic: "greetings", Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	app.Hub.Register <- crx

	time.Sleep(2 * time.Millisecond)

	// check hubstats
	if len(app.Hub.Hub.Clients) != 1 {
		t.Errorf("Wrong number of clients registered to hub wanted/got %d/%d", 1, len(app.Hub.Hub.Clients))
	}

	// server to action the handler under test
	router := mux.NewRouter()
	router.HandleFunc("/ws/{feed}", http.HandlerFunc(app.handleWs))

	s := httptest.NewServer(router)
	defer s.Close()

	time.Sleep(2 * time.Millisecond)

	// test harness on the sending side
	r := reconws.New()
	urlStr := "ws" + strings.TrimPrefix(s.URL, "http") + "/ws/greetings"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Reconnect(ctx, urlStr)

	time.Sleep(1 * time.Millisecond)

	//messages we will use
	greeting := []byte("hello") // from websocket to hub client, via agg
	reply := []byte("howdy")    //from hub client to websocket, via agg

	// set up the receiver in a separate goroutine to avoid the hub deleting this client

	go func() {
		//did message get to hub client?
		select {
		case <-time.After(10 * time.Millisecond):
			t.Error("timed out")
		case msg, ok := <-crx.Send:
			if ok {
				if !bytes.Equal(msg.Data, greeting) {
					t.Errorf("Greeting content unexpected; got/wanted %v/%v\n", string(msg.Data), string(greeting))
				}
				//reply
				crx.Hub.Broadcast <- hub.Message{Sender: *crx, Sent: time.Now(), Data: reply, Type: websocket.TextMessage}

			} else {
				t.Error("channel not ok") //this test seems sensitive to timing off the sleeps, registration delay?
			}
		}

		// did reply get to websocket client?
		select {
		case <-time.After(10 * time.Millisecond):
			t.Error("timed out")
		case msg, ok := <-r.In:
			if ok {
				if !bytes.Equal(msg.Data, reply) {
					t.Errorf("Reply content unexpected; got/wanted %v/%v\n", string(msg.Data), string(reply))
				}
			} else {
				t.Error("channel not ok")
			}
		}
	}()

	time.Sleep(1 * time.Millisecond)

	// test - send a message :-
	go func() {
		m := &reconws.WsMessage{Data: greeting, Type: websocket.TextMessage}
		r.Out <- *m
	}()

	// hang on long enough for both timeouts in the anonymous goroutine
	time.Sleep(30 * time.Millisecond)

}

// this test only shows that the httptest server is working ok
func TestHandleWsEcho(t *testing.T) {

	r := reconws.New()

	// Create test server with the echo handler.
	s := httptest.NewServer(http.HandlerFunc(echo))
	defer s.Close()

	// Convert http://127.0.0.1 to ws://127.0.0.
	urlStr := "ws" + strings.TrimPrefix(s.URL, "http")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Reconnect(ctx, urlStr)

	payload := []byte("Hello")
	mtype := int(websocket.TextMessage)

	r.Out <- reconws.WsMessage{Data: payload, Type: mtype}

	reply := <-r.In

	if !bytes.Equal(reply.Data, payload) {
		t.Errorf("Got unexpected response: %s, wanted %s\n", reply.Data, payload)
	}

}

var testUpgrader = websocket.Upgrader{}

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := testUpgrader.Upgrade(w, r, nil)
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
