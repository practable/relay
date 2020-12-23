package crossbar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"
	"github.com/phayes/freeport"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/timdrysdale/reconws"
)

func MakeTestToken(audience string, lifetime int64, secret string) (string, error) {

	now := time.Now().Unix()
	later := now + lifetime
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud": audience,
		"iat": now,
		"nbf": now,
		"exp": later,
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(secret))

	return tokenString, err

}

func MakeExpiredTestToken(audience string, secret string) (string, error) {

	now := time.Now().Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud": audience,
		"iat": now - 200,
		"nbf": now - 100,
		"exp": now - 5,
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(secret))

	return tokenString, err

}

func MakeTooEarlyTestToken(audience string, secret string) (string, error) {

	now := time.Now().Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud": audience,
		"iat": now,
		"nbf": now + 100,
		"exp": now + 200,
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(secret))

	return tokenString, err

}

func TestTooEarlyForAuth(t *testing.T) {
	//log.SetLevel(log.TraceLevel)
	suppressLog()
	defer displayLog()

	//Todo - add support for httptest https://stackoverflow.com/questions/40786526/resetting-http-handlers-in-golang-for-unit-testing
	http.DefaultServeMux = new(http.ServeMux)

	// setup crossbar on local (free) port

	closed := make(chan struct{})
	var wg sync.WaitGroup

	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	addr := ":" + strconv.Itoa(port)
	route := "ws://127.0.0.1" + addr
	secret := "asldjflkasjdflkj13094809asdfhkj13"
	config := Config{
		Addr:     addr,
		Audience: route,
		Secret:   secret,
	}

	wg.Add(1)

	go Crossbar(config, closed, &wg)

	time.Sleep(10 * time.Millisecond)

	// set up test server and two clients

	ctx, cancel := context.WithCancel(context.Background())

	serverEndPoint := "/in/some/location"
	us := route + serverEndPoint

	earlyToken, err := MakeTooEarlyTestToken(us, secret)
	assert.NoError(t, err)

	s := reconws.New()

	go s.Reconnect(ctx, us)

	timeout := 100 * time.Millisecond

	time.Sleep(timeout)

	// do authorisation
	mtype := websocket.TextMessage

	s.Out <- reconws.WsMessage{Data: []byte(earlyToken), Type: mtype}

	expectedServerReply, err := json.Marshal(AuthMessage{
		Topic:      serverEndPoint,
		Token:      earlyToken,
		Authorised: false,
		Reason:     "Error reading token Token is not valid yet",
	})

	_ = expectOneSlice(s.In, expectedServerReply, timeout, t)

	time.Sleep(timeout)

	cancel()

	time.Sleep(timeout)

	close(closed)

	wg.Wait()

}

func TestTooLateForAuth(t *testing.T) {
	//log.SetLevel(log.TraceLevel)
	suppressLog()
	defer displayLog()

	//Todo - add support for httptest https://stackoverflow.com/questions/40786526/resetting-http-handlers-in-golang-for-unit-testing
	http.DefaultServeMux = new(http.ServeMux)

	// setup Crossbar on local (free) port

	closed := make(chan struct{})
	var wg sync.WaitGroup

	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	addr := ":" + strconv.Itoa(port)
	route := "ws://127.0.0.1" + addr
	secret := "asldjflkasjdflkj13094809asdfhkj13"
	config := Config{
		Addr:     addr,
		Audience: route,
		Secret:   secret,
	}

	wg.Add(1)

	go Crossbar(config, closed, &wg)

	time.Sleep(10 * time.Millisecond)

	// set up test server and two clients

	ctx, cancel := context.WithCancel(context.Background())

	serverEndPoint := "/in/some/location"
	us := route + serverEndPoint

	lateToken, err := MakeExpiredTestToken(us, secret)
	assert.NoError(t, err)

	s := reconws.New()

	go s.Reconnect(ctx, us)

	timeout := 100 * time.Millisecond

	time.Sleep(timeout)

	// do authorisation
	mtype := websocket.TextMessage

	s.Out <- reconws.WsMessage{Data: []byte(lateToken), Type: mtype}

	expectedServerReply, err := json.Marshal(AuthMessage{
		Topic:      serverEndPoint,
		Token:      lateToken,
		Authorised: false,
		Reason:     "Error reading token Token is expired",
	})

	_ = expectOneSlice(s.In, expectedServerReply, timeout, t)

	time.Sleep(timeout)

	cancel()

	time.Sleep(timeout)

	close(closed)

	wg.Wait()

}

func TestBadServerAuth(t *testing.T) {
	//log.SetLevel(log.TraceLevel)
	suppressLog()
	defer displayLog()

	//Todo - add support for httptest https://stackoverflow.com/questions/40786526/resetting-http-handlers-in-golang-for-unit-testing
	http.DefaultServeMux = new(http.ServeMux)

	// setup Crossbar on local (free) port

	closed := make(chan struct{})
	var wg sync.WaitGroup

	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	addr := ":" + strconv.Itoa(port)
	route := "ws://127.0.0.1" + addr
	secret := "asldjflkasjdflkj13094809asdfhkj13"
	config := Config{
		Addr:     addr,
		Audience: route,
		Secret:   secret,
	}

	wg.Add(1)

	go Crossbar(config, closed, &wg)

	time.Sleep(10 * time.Millisecond)

	// set up test server and two clients

	ctx, cancel := context.WithCancel(context.Background())

	clientEndPoint := "/out/some/location"
	serverEndPoint := "/in/some/location"
	uc := route + clientEndPoint
	us := route + serverEndPoint

	var lifetime int64 = 999999
	ctoken, err := MakeTestToken(uc, lifetime, secret)

	assert.NoError(t, err)
	badstoken, err := MakeTestToken(uc, lifetime, secret) //deliberately wrong routing (should have .../in/... not .../out/...)

	assert.NoError(t, err)

	c0 := reconws.New()
	s := reconws.New()

	go c0.Reconnect(ctx, uc)
	go s.Reconnect(ctx, us)

	timeout := 100 * time.Millisecond

	time.Sleep(timeout)

	// do authorisation
	mtype := websocket.TextMessage

	c0.Out <- reconws.WsMessage{Data: []byte(ctoken), Type: mtype}
	s.Out <- reconws.WsMessage{Data: []byte(badstoken), Type: mtype}

	expectedClientReply, err := json.Marshal(AuthMessage{
		Topic:      clientEndPoint,
		Token:      ctoken,
		Authorised: true,
		Reason:     "ok",
	})

	assert.NoError(t, err)

	_ = expectOneSlice(c0.In, expectedClientReply, timeout, t)

	expectedServerReply, err := json.Marshal(AuthMessage{
		Topic:      serverEndPoint,
		Token:      badstoken,
		Authorised: false,
		Reason:     fmt.Sprintf("Denied - not permitted to access %s with token for %s", us, uc),
	})

	_ = expectOneSlice(s.In, expectedServerReply, timeout, t)

	payload0 := []byte("Hello from client0")

	c0.Out <- reconws.WsMessage{Data: payload0, Type: mtype}

	expectNoMsg(s.In, timeout, t)  //should not see message from any client
	expectNoMsg(c0.In, timeout, t) //should not see message from other client

	// broadcast from the server - note that due to reconnecting websocket reconws being
	// used in this test, the connection will have closed after the last failed auth,
	// so it will connect again, and this message will count as a "first" connection,
	// so the broadcast message will be treated as the
	// access token, and the server will fail to authorise (again)
	// TODO we will want to build servers and clients that are polite about reconnecting
	// after denial! (hence the value in sending the denial message)

	broadcast0 := []byte("First broadcast from server")

	s.Out <- reconws.WsMessage{Data: broadcast0, Type: websocket.BinaryMessage}

	expectNoMsg(c0.In, timeout, t) //should not see message from unauthed server

	expectedServerReply, err = json.Marshal(AuthMessage{
		Topic:      serverEndPoint,
		Token:      string(broadcast0),
		Authorised: false,
		Reason:     "Error reading token token contains an invalid number of segments",
	})

	_ = expectOneSlice(s.In, expectedServerReply, timeout, t)

	time.Sleep(timeout)

	cancel()

	time.Sleep(timeout)

	close(closed)

	wg.Wait()

}

func TestBadClientAuth(t *testing.T) {
	//log.SetLevel(log.TraceLevel)
	suppressLog()
	defer displayLog()

	//Todo - add support for httptest https://stackoverflow.com/questions/40786526/resetting-http-handlers-in-golang-for-unit-testing
	http.DefaultServeMux = new(http.ServeMux)

	// setup Crossbar on local (free) port

	closed := make(chan struct{})
	var wg sync.WaitGroup

	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	addr := ":" + strconv.Itoa(port)
	route := "ws://127.0.0.1" + addr
	secret := "asldjflkasjdflkj13094809asdfhkj13"
	config := Config{
		Addr:     addr,
		Audience: route,
		Secret:   secret,
	}

	wg.Add(1)

	go Crossbar(config, closed, &wg)

	time.Sleep(10 * time.Millisecond)

	// set up test server and two clients

	ctx, cancel := context.WithCancel(context.Background())

	clientEndPoint := "/out/some/location"
	serverEndPoint := "/in/some/location"
	uc := route + clientEndPoint
	us := route + serverEndPoint
	ubad := route + "/bad/routing"

	var lifetime int64 = 999999
	ctoken, err := MakeTestToken(uc, lifetime, secret)

	badctoken, err := MakeTestToken(ubad, lifetime, secret)

	assert.NoError(t, err)
	stoken, err := MakeTestToken(us, lifetime, secret)

	assert.NoError(t, err)

	c0 := reconws.New()
	c1 := reconws.New()
	s := reconws.New()

	go c0.Reconnect(ctx, uc)
	go c1.Reconnect(ctx, uc)
	go s.Reconnect(ctx, us)

	timeout := 100 * time.Millisecond

	time.Sleep(timeout)

	// do authorisation
	mtype := websocket.TextMessage

	c0.Out <- reconws.WsMessage{Data: []byte(ctoken), Type: mtype}
	c1.Out <- reconws.WsMessage{Data: []byte(badctoken), Type: mtype}
	s.Out <- reconws.WsMessage{Data: []byte(stoken), Type: mtype}

	expectedClientReply, err := json.Marshal(AuthMessage{
		Topic:      clientEndPoint,
		Token:      ctoken,
		Authorised: true,
		Reason:     "ok",
	})

	assert.NoError(t, err)

	_ = expectOneSlice(c0.In, expectedClientReply, timeout, t)

	expectedBadClientReply, err := json.Marshal(AuthMessage{
		Topic:      clientEndPoint,
		Token:      badctoken,
		Authorised: false,
		Reason:     fmt.Sprintf("Denied - not permitted to access %s with token for %s", uc, ubad),
	})

	assert.NoError(t, err)

	_ = expectOneSlice(c1.In, expectedBadClientReply, timeout, t)

	expectedServerReply, err := json.Marshal(AuthMessage{
		Topic:      serverEndPoint,
		Token:      stoken,
		Authorised: true,
		Reason:     "ok",
	})

	_ = expectOneSlice(s.In, expectedServerReply, timeout, t)

	payload0 := []byte("Hello from client0")
	payload1 := []byte("Hello from client1")

	c0.Out <- reconws.WsMessage{Data: payload0, Type: mtype}
	c1.Out <- reconws.WsMessage{Data: payload1, Type: mtype}

	expectNoMsg(s.In, timeout, t)  //should not see message from any client
	expectNoMsg(c0.In, timeout, t) //should not see message from other client
	expectNoMsg(c1.In, timeout, t) //should not see message from other client

	// broadcast from the server

	broadcast0 := []byte("First broadcast from server")
	broadcast1 := []byte("Second broadcast from server")

	s.Out <- reconws.WsMessage{Data: broadcast0, Type: websocket.BinaryMessage}

	_ = expectOneSlice(c0.In, broadcast0, timeout, t)

	expectNoMsg(c1.In, timeout, t) //should not see message from unauthed client
	expectNoMsg(s.In, timeout, t)  //no echo

	s.Out <- reconws.WsMessage{Data: broadcast1, Type: websocket.BinaryMessage}

	_ = expectOneSlice(c0.In, broadcast1, timeout, t)

	expectNoMsg(c1.In, timeout, t) //should not see message from unauthed client

	expectNoMsg(s.In, timeout, t)  //no echo
	expectNoMsg(c0.In, timeout, t) //only expecting two messages
	expectNoMsg(c1.In, timeout, t) //no third message expected

	time.Sleep(timeout)

	cancel()

	time.Sleep(timeout)

	close(closed)

	wg.Wait()

}

func TestAuth(t *testing.T) {
	//log.SetLevel(log.TraceLevel)
	suppressLog()
	defer displayLog()

	//Todo - add support for httptest https://stackoverflow.com/questions/40786526/resetting-http-handlers-in-golang-for-unit-testing
	http.DefaultServeMux = new(http.ServeMux)

	// setup Crossbar on local (free) port

	closed := make(chan struct{})
	var wg sync.WaitGroup

	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	addr := ":" + strconv.Itoa(port)
	route := "ws://127.0.0.1" + addr
	secret := "asldjflkasjdflkj13094809asdfhkj13"
	config := Config{
		Addr:     addr,
		Audience: route,
		Secret:   secret,
	}

	wg.Add(1)

	go Crossbar(config, closed, &wg)

	time.Sleep(10 * time.Millisecond)

	// set up test server and two clients

	ctx, cancel := context.WithCancel(context.Background())

	clientEndPoint := "/out/some/location"
	serverEndPoint := "/in/some/location"
	uc := route + clientEndPoint
	us := route + serverEndPoint

	var lifetime int64 = 999999
	ctoken, err := MakeTestToken(uc, lifetime, secret)

	assert.NoError(t, err)
	stoken, err := MakeTestToken(us, lifetime, secret)

	assert.NoError(t, err)

	c0 := reconws.New()
	c1 := reconws.New()
	s := reconws.New()

	go c0.Reconnect(ctx, uc)
	go c1.Reconnect(ctx, uc)
	go s.Reconnect(ctx, us)

	timeout := 50 * time.Millisecond

	time.Sleep(timeout)

	// do authorisation
	mtype := websocket.TextMessage

	c0.Out <- reconws.WsMessage{Data: []byte(ctoken), Type: mtype}
	c1.Out <- reconws.WsMessage{Data: []byte(ctoken), Type: mtype}
	s.Out <- reconws.WsMessage{Data: []byte(stoken), Type: mtype}

	expectedClientReply, err := json.Marshal(AuthMessage{
		Topic:      clientEndPoint,
		Token:      ctoken,
		Authorised: true,
		Reason:     "ok",
	})

	assert.NoError(t, err)

	_ = expectOneSlice(c0.In, expectedClientReply, timeout, t)
	_ = expectOneSlice(c1.In, expectedClientReply, timeout, t)

	expectedServerReply, err := json.Marshal(AuthMessage{
		Topic:      serverEndPoint,
		Token:      stoken,
		Authorised: true,
		Reason:     "ok",
	})

	_ = expectOneSlice(s.In, expectedServerReply, timeout, t)

	payload0 := []byte("Hello from client0")
	payload1 := []byte("Hello from client1")

	c0.Out <- reconws.WsMessage{Data: payload0, Type: mtype}
	c1.Out <- reconws.WsMessage{Data: payload1, Type: mtype}

	expectNoMsg(s.In, timeout, t)  //should not see message from any client
	expectNoMsg(c0.In, timeout, t) //should not see message from other client
	expectNoMsg(c1.In, timeout, t) //should not see message from other client

	// broadcast from the server

	broadcast0 := []byte("First broadcast from server")
	broadcast1 := []byte("Second broadcast from server")

	s.Out <- reconws.WsMessage{Data: broadcast0, Type: websocket.BinaryMessage}

	_ = expectOneSlice(c0.In, broadcast0, timeout, t)
	_ = expectOneSlice(c1.In, broadcast0, timeout, t)
	expectNoMsg(s.In, timeout, t) //no echo

	s.Out <- reconws.WsMessage{Data: broadcast1, Type: websocket.BinaryMessage}

	_ = expectOneSlice(c0.In, broadcast1, timeout, t)
	_ = expectOneSlice(c1.In, broadcast1, timeout, t)
	expectNoMsg(s.In, timeout, t)  //no echo
	expectNoMsg(c0.In, timeout, t) //only expecting two messages
	expectNoMsg(c1.In, timeout, t) //no third message expected

	time.Sleep(timeout)

	cancel()

	time.Sleep(timeout)

	close(closed)

	wg.Wait()

}

func TestCrossbarUniDirectionalMessaging(t *testing.T) {

	suppressLog()
	defer displayLog()

	//Todo - add support for httptest https://stackoverflow.com/questions/40786526/resetting-http-handlers-in-golang-for-unit-testing
	http.DefaultServeMux = new(http.ServeMux)

	// setup Crossbar on local (free) port

	closed := make(chan struct{})
	var wg sync.WaitGroup

	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	addr := ":" + strconv.Itoa(port)
	config := Config{
		Addr: addr,
	}
	wg.Add(1)
	go Crossbar(config, closed, &wg)

	time.Sleep(10 * time.Millisecond)

	// set up test server and two clients

	ctx, cancel := context.WithCancel(context.Background())

	uc := "ws://127.0.0.1" + addr + "/out/some/location"
	us := "ws://127.0.0.1" + addr + "/in/some/location"

	c0 := reconws.New()
	c1 := reconws.New()
	s := reconws.New()

	go c0.Reconnect(ctx, uc)
	go c1.Reconnect(ctx, uc)
	go s.Reconnect(ctx, us)

	timeout := 50 * time.Millisecond

	time.Sleep(timeout)

	payload0 := []byte("Hello from client0")
	payload1 := []byte("Hello from client1")

	mtype := websocket.TextMessage

	c0.Out <- reconws.WsMessage{Data: payload0, Type: mtype}
	c1.Out <- reconws.WsMessage{Data: payload1, Type: mtype}

	expectNoMsg(s.In, timeout, t)  //should not see message from any client
	expectNoMsg(c0.In, timeout, t) //should not see message from other client
	expectNoMsg(c1.In, timeout, t) //should not see message from other client

	// broadcast from the server

	broadcast0 := []byte("First broadcast from server")
	broadcast1 := []byte("Second broadcast from server")

	s.Out <- reconws.WsMessage{Data: broadcast0, Type: websocket.BinaryMessage}

	_ = expectOneSlice(c0.In, broadcast0, timeout, t)
	_ = expectOneSlice(c1.In, broadcast0, timeout, t)
	expectNoMsg(s.In, timeout, t) //no echo

	s.Out <- reconws.WsMessage{Data: broadcast1, Type: websocket.BinaryMessage}

	_ = expectOneSlice(c0.In, broadcast1, timeout, t)
	_ = expectOneSlice(c1.In, broadcast1, timeout, t)
	expectNoMsg(s.In, timeout, t)  //no echo
	expectNoMsg(c0.In, timeout, t) //only expecting two messages
	expectNoMsg(c1.In, timeout, t) //no third message expected

	time.Sleep(timeout)

	cancel()

	time.Sleep(timeout)

	close(closed)

	wg.Wait()

}

func TestCrossbarBidirectionalMessaging(t *testing.T) {

	suppressLog()
	defer displayLog()

	//TODO - add support for httptest https://stackoverflow.com/questions/40786526/resetting-http-handlers-in-golang-for-unit-testing
	http.DefaultServeMux = new(http.ServeMux)

	// setup Crossbar on local (free) port

	closed := make(chan struct{})
	var wg sync.WaitGroup

	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	addr := ":" + strconv.Itoa(port)

	config := Config{
		Addr: addr,
	}

	wg.Add(1)
	go Crossbar(config, closed, &wg)

	time.Sleep(10 * time.Millisecond)

	// set up test server and two clients

	ctx, cancel := context.WithCancel(context.Background())

	uc := "ws://127.0.0.1" + addr + "/bi/some/location"
	us := "ws://127.0.0.1" + addr + "/bi/some/location"

	c0 := reconws.New()
	c1 := reconws.New()
	s := reconws.New()

	go c0.Reconnect(ctx, uc)
	go c1.Reconnect(ctx, uc)
	go s.Reconnect(ctx, us)

	timeout := 50 * time.Millisecond

	time.Sleep(timeout)

	payload0 := []byte("Hello from client0")
	payload1 := []byte("Hello from client1")

	mtype := websocket.TextMessage

	//this message goes to s and c1
	c0.Out <- reconws.WsMessage{Data: payload0, Type: mtype}
	_ = expectOneSlice(s.In, payload0, timeout, t)
	_ = expectOneSlice(c1.In, payload0, timeout, t)
	expectNoMsg(c0.In, timeout, t) //should not see message from self

	//this message goes to s and c0
	c1.Out <- reconws.WsMessage{Data: payload1, Type: mtype}
	_ = expectOneSlice(s.In, payload1, timeout, t)
	_ = expectOneSlice(c0.In, payload1, timeout, t)
	expectNoMsg(c1.In, timeout, t) //should not see message from self

	// the server should get each message only once
	expectNoMsg(s.In, timeout, t)

	// broadcast from the server

	broadcast0 := []byte("First broadcast from server")
	broadcast1 := []byte("Second broadcast from server")

	s.Out <- reconws.WsMessage{Data: broadcast0, Type: websocket.BinaryMessage}

	_ = expectOneSlice(c0.In, broadcast0, timeout, t)
	_ = expectOneSlice(c1.In, broadcast0, timeout, t)
	expectNoMsg(s.In, timeout, t) //no echo

	s.Out <- reconws.WsMessage{Data: broadcast1, Type: websocket.BinaryMessage}

	_ = expectOneSlice(c0.In, broadcast1, timeout, t)
	_ = expectOneSlice(c1.In, broadcast1, timeout, t)
	expectNoMsg(s.In, timeout, t)  //no echo
	expectNoMsg(c0.In, timeout, t) //only expecting two messages
	expectNoMsg(c1.In, timeout, t) //no third message expected

	time.Sleep(timeout)

	cancel()

	time.Sleep(timeout)

	close(closed)

	wg.Wait()

}

func expectNoMsg(channel chan reconws.WsMessage, timeout time.Duration, t *testing.T) {

	select {
	case <-time.After(timeout):
		return //we are expecting to timeout, this is good
	case msg, ok := <-channel:
		if ok {
			t.Errorf("Receieved unexpected message %s", msg.Data)
		} else {
			//just a channel problem, not an unexpected message
		}
	}
}

func expectOneSlice(channel chan reconws.WsMessage, expected []byte, timeout time.Duration, t *testing.T) []byte {

	var receivedSlice []byte

	select {
	case <-time.After(timeout):
		t.Errorf("timeout receiving message (expected %s)", expected)
	case msg, ok := <-channel:
		if ok && len(expected) > 0 {
			receivedSlice = msg.Data
			if bytes.Compare(receivedSlice, expected) != 0 {
				t.Errorf("Messages don't match: Want: %s\nGot : %s\n", expected, receivedSlice)
			}
		} else if !ok {
			t.Error("Channel problem")
		} else { //for the case we didn't know in advance the reply type ....
			// use this only for debugging tests
			receivedSlice = msg.Data
		}
	}
	return receivedSlice
}
