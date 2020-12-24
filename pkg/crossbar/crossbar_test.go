package crossbar

import (
	"bytes"
	"context"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/phayes/freeport"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/timdrysdale/reconws"
	"github.com/timdrysdale/relay/pkg/permission"
	"github.com/timdrysdale/relay/pkg/ttlcode"
)

var testSession = "/session/20fd9a71-2248-4f60-89e3-5d5bb2e78e09"
var wrongSession = "/session/90fd9a71-2247-5f61-76e5-4d5cc2e34e0f"

var testShell = "/shell/4cb1e5ff-5f48-49f7-bdee-a5a58d331293"
var wrongConnection = "/in/bc0761b6-ca00-49fc-aa91-122234d936b4"

var testScopesRW = []string{"read", "write"}
var testScopesR = []string{"read"}
var testScopesW = []string{"write"}
var testScopesNone = []string{}

var wrongAudience = "foo.example.io"

var testAudience = "ws://127.0.0.1"

var testLifetime = int64(5)
var testBuffer = int64(1)

func MakeDefaultTestToken(audience string) permission.Token {
	return MakeTestToken(audience, testSession, testScopesRW, testLifetime)
}

func MakeTestToken(audience, topic string, scopes []string, lifetime int64) permission.Token {
	now := time.Now().Unix() - testBuffer
	later := now + lifetime + testBuffer
	return permission.NewToken(audience, topic, scopes, now, now, later)
}

func MakeExpiredTestToken(audience string) permission.Token {
	earlier := time.Now().Unix() - 100
	evenEarlier := earlier - 200
	return permission.NewToken(audience, testSession, testScopesRW, evenEarlier, evenEarlier, earlier)
}

func MakeTooEarlyTestToken(audience string) permission.Token {
	later := time.Now().Unix() + 100
	evenLater := later + 200
	return permission.NewToken(audience, testSession, testScopesRW, later, later, evenLater)
}

func MakeScopeTestToken(audience string, scopes []string) permission.Token {
	return MakeTestToken(audience, testSession, scopes, testLifetime)
}

func MakeWrongAudienceTestToken() permission.Token {
	return MakeTestToken(wrongAudience, testSession, testScopesRW, testLifetime)
}

func MakeWrongSessionTestToken(audience string) permission.Token {
	return MakeTestToken(audience, wrongSession, testScopesRW, testLifetime)
}

func MakeUnsupportedConnectionTestToken(audience string) permission.Token {
	return MakeTestToken(audience, wrongConnection, testScopesRW, testLifetime)
}

func getTestAudience(port int) string {
	return testAudience + ":" + strconv.Itoa(port)
}

func TestCanAuthWithCode(t *testing.T) {

	defer debug(false)()

	//Todo - add support for httptest https://stackoverflow.com/questions/40786526/resetting-http-handlers-in-golang-for-unit-testing
	http.DefaultServeMux = new(http.ServeMux)

	// setup crossbar on local (free) port
	closed := make(chan struct{})
	var wg sync.WaitGroup

	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	audience := getTestAudience(port)
	cs := ttlcode.NewDefaultCodeStore()
	config := Config{
		Listen:    port,
		Audience:  audience,
		CodeStore: cs,
	}

	wg.Add(1)

	go Crossbar(config, closed, &wg)

	time.Sleep(10 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())

	token := MakeDefaultTestToken(audience)

	code0 := cs.SubmitToken(permission.ConvertToJWT(token))
	code1 := cs.SubmitToken(permission.ConvertToJWT(token))

	serverEndPoint := testSession + "?code="
	us0 := audience + serverEndPoint + code0
	us1 := audience + serverEndPoint + code1

	timeout := 100 * time.Millisecond
	time.Sleep(timeout)

	s0 := reconws.New()
	go s0.Reconnect(ctx, us0)

	s1 := reconws.New()
	go s1.Reconnect(ctx, us1)

	// do authorisation
	mtype := websocket.TextMessage

	data := []byte("foo")
	s0.Out <- reconws.WsMessage{Data: data, Type: mtype}

	_ = expectOneSlice(s1.In, data, timeout, t)

	time.Sleep(timeout)

	cancel()

	time.Sleep(timeout)

	close(closed)

	wg.Wait()

}

func TestSingleUseCode(t *testing.T) {

	defer debug(false)()

	//Todo - add support for httptest https://stackoverflow.com/questions/40786526/resetting-http-handlers-in-golang-for-unit-testing
	http.DefaultServeMux = new(http.ServeMux)

	// setup crossbar on local (free) port
	closed := make(chan struct{})
	var wg sync.WaitGroup

	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	audience := getTestAudience(port)
	cs := ttlcode.NewDefaultCodeStore()
	config := Config{
		Listen:    port,
		Audience:  audience,
		CodeStore: cs,
	}

	wg.Add(1)

	go Crossbar(config, closed, &wg)

	time.Sleep(10 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())

	token := MakeDefaultTestToken(audience)

	code0 := cs.SubmitToken(permission.ConvertToJWT(token))
	code1 := cs.SubmitToken(permission.ConvertToJWT(token))

	serverEndPoint := testSession + "?code="
	us0 := audience + serverEndPoint + code0
	us1 := audience + serverEndPoint + code1

	timeout := 100 * time.Millisecond
	time.Sleep(timeout)

	s0 := reconws.New()
	go s0.Reconnect(ctx, us0)

	s1 := reconws.New()
	go s1.Reconnect(ctx, us1)

	s2 := reconws.New()
	go s2.Reconnect(ctx, us1)

	// do authorisation
	mtype := websocket.TextMessage

	data := []byte("foo")
	s0.Out <- reconws.WsMessage{Data: data, Type: mtype}

	// can't force which client gets the connection approved
	// but ONLY one should
	select {
	case msg := <-s1.In:
		assert.Equal(t, data, msg.Data)
		expectNoMsg(s2.In, timeout, t)
	case msg := <-s2.In:
		assert.Equal(t, data, msg.Data)
		expectNoMsg(s1.In, timeout, t)
	}

	time.Sleep(timeout)

	cancel()

	time.Sleep(timeout)

	close(closed)

	wg.Wait()

}

func TestScopesEnforcedRW(t *testing.T) {

	defer debug(false)()

	//Todo - add support for httptest https://stackoverflow.com/questions/40786526/resetting-http-handlers-in-golang-for-unit-testing
	http.DefaultServeMux = new(http.ServeMux)

	// setup crossbar on local (free) port
	closed := make(chan struct{})
	var wg sync.WaitGroup

	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	audience := getTestAudience(port)
	cs := ttlcode.NewDefaultCodeStore()
	config := Config{
		Listen:    port,
		Audience:  audience,
		CodeStore: cs,
	}

	wg.Add(1)

	go Crossbar(config, closed, &wg)

	time.Sleep(10 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())

	token := MakeDefaultTestToken(audience)

	code0 := cs.SubmitToken(permission.ConvertToJWT(token))
	code1 := cs.SubmitToken(permission.ConvertToJWT(token))

	serverEndPoint := testSession + "?code="
	us0 := audience + serverEndPoint + code0
	us1 := audience + serverEndPoint + code1

	timeout := 100 * time.Millisecond
	time.Sleep(timeout)

	s0 := reconws.New()
	go s0.Reconnect(ctx, us0)

	s1 := reconws.New()
	go s1.Reconnect(ctx, us1)

	// do authorisation
	mtype := websocket.TextMessage

	data := []byte("foo")
	s0.Out <- reconws.WsMessage{Data: data, Type: mtype}

	_ = expectOneSlice(s1.In, data, timeout, t)

	time.Sleep(timeout)

	data = []byte("bar")

	s1.Out <- reconws.WsMessage{Data: data, Type: mtype}

	_ = expectOneSlice(s0.In, data, timeout, t)

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
