package crossbar_test

import (
	"bufio"
	"bytes"
	"context"
	"net/http"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/phayes/freeport"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/timdrysdale/reconws"
	"github.com/timdrysdale/relay/pkg/crossbar"
	"github.com/timdrysdale/relay/pkg/permission"
	"github.com/timdrysdale/relay/pkg/ttlcode"
)

// NOTE don't use reconws.Reconnect for production clients anymore;
// it does NOT understand the use of auth codes
// use Dial instead.

func MakeTestToken(audience, topic string, scopes []string, lifetime int64) permission.Token {
	begin := time.Now().Unix() - 1 //ensure it's in the past
	end := begin + lifetime
	return permission.NewToken(audience, topic, scopes, begin, begin, end)
}

func TestExternalCrossbar(t *testing.T) {

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

	// Setup crossbar

	http.DefaultServeMux = new(http.ServeMux)

	// setup crossbar on local (free) port
	closed := make(chan struct{})
	var wg sync.WaitGroup

	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	audience := "ws://127.0.0.1:" + strconv.Itoa(port)
	cs := ttlcode.NewDefaultCodeStore()
	config := crossbar.Config{
		Listen:    port,
		Audience:  audience,
		CodeStore: cs,
	}

	wg.Add(1)
	go crossbar.Crossbar(config, closed, &wg)
	// safety margin to get crossbar running
	time.Sleep(time.Second)

	var timeout = 100 * time.Millisecond

	// Start tests

	// *** TestCanConnectWithValidCode ***
	// these parameters reused by following tests:

	ctx, cancel := context.WithCancel(context.Background())

	session := "/session/20fd9a71-2248-4f60-89e3-5d5bb2e78e09"
	scopes := []string{"read", "write"}

	token := MakeTestToken(audience, session, scopes, 5)

	code0 := cs.SubmitToken(permission.ConvertToJWT(token))
	code1 := cs.SubmitToken(permission.ConvertToJWT(token))

	s0 := reconws.New()
	go s0.Dial(ctx, audience+session+"?code="+code0)

	s1 := reconws.New()
	go s1.Dial(ctx, audience+session+"?code="+code1)

	time.Sleep(timeout)

	data := []byte("foo")

	s0.Out <- reconws.WsMessage{Data: data, Type: websocket.TextMessage}

	select {
	case msg := <-s1.In:
		assert.Equal(t, data, msg.Data)
		t.Logf("TestCanConnectWithValidCode...PASS\n")
	case <-time.After(timeout):
		t.Fatal("TestCanConnectWithValidCode...FAIL")
	}
	cancel()
	time.Sleep(timeout)

	// *** TestCannotConnectWithReusedCode ***
	// try the last test again, without getting new codes

	ctx, cancel = context.WithCancel(context.Background())

	go s0.Dial(ctx, audience+session+"?code="+code0)
	go s1.Dial(ctx, audience+session+"?code="+code1)

	time.Sleep(timeout)

	s0.Out <- reconws.WsMessage{Data: data, Type: websocket.TextMessage}

	select {
	case <-s1.In:
		t.Fatal("TestCannotConnectWithReusedCode...FAIL")
	case <-time.After(timeout):
		t.Log("TestCannotConnectWithReusedCode...PASS")
	}
	cancel()
	time.Sleep(timeout)

	// *** TestEnforceScopesRWToRW

	// try the last test again, getting new codes, and replying

	code0 = cs.SubmitToken(permission.ConvertToJWT(token))
	code1 = cs.SubmitToken(permission.ConvertToJWT(token))

	ctx, cancel = context.WithCancel(context.Background())

	go s0.Dial(ctx, audience+session+"?code="+code0)
	go s1.Dial(ctx, audience+session+"?code="+code1)

	time.Sleep(timeout)

	data = []byte("ping")
	s0.Out <- reconws.WsMessage{Data: data, Type: websocket.TextMessage}
	select {
	case msg := <-s1.In:
		assert.Equal(t, data, msg.Data)
	case <-time.After(timeout):
		t.Fatal("TestEnforceScopesRWToRW...FAIL")
	}

	data = []byte("pong")
	s1.Out <- reconws.WsMessage{Data: data, Type: websocket.TextMessage}
	select {
	case msg := <-s0.In:
		assert.Equal(t, data, msg.Data)
		t.Log("TestEnforceScopesRWToRW...PASS")
	case <-time.After(timeout):
		t.Fatal("TestEnforceScopesRWToRW...FAIL")
	}

	cancel()
	time.Sleep(timeout)

	// *** TestEnforceScopesRWToR

	// try the last test again, getting new codes, and replying (blocked!)
	scopes = []string{"read"}
	tokenReadOnly := MakeTestToken(audience, session, scopes, 5)

	code0 = cs.SubmitToken(permission.ConvertToJWT(token))
	code1 = cs.SubmitToken(permission.ConvertToJWT(tokenReadOnly))

	ctx, cancel = context.WithCancel(context.Background())

	go s0.Dial(ctx, audience+session+"?code="+code0)
	go s1.Dial(ctx, audience+session+"?code="+code1)

	time.Sleep(timeout)

	data = []byte("ping")
	s0.Out <- reconws.WsMessage{Data: data, Type: websocket.TextMessage}
	select {
	case msg := <-s1.In:
		assert.Equal(t, data, msg.Data)
	case <-time.After(timeout):
		t.Fatal("TestEnforceScopesRWToROnly...FAIL")
	}

	data = []byte("nopong")
	// Send a message - should be silently ignored by crossbar
	s1.Out <- reconws.WsMessage{Data: data, Type: websocket.TextMessage}
	select {
	case <-s0.In:
		t.Fatal("TestEnforceScopesRWToROnly...FAIL")
	case <-time.After(timeout):
		t.Log("TestEnforceScopesRWToROnly...PASS")
	}

	cancel()
	time.Sleep(timeout)

	// *** TestEnforceAudience

	// reader connects with a token intended for the same session on another server
	// should NOT receive message!
	scopes = []string{"read"}
	tokenWrongAudience := MakeTestToken("ws://wrong.server.io", session, scopes, 5)

	code0 = cs.SubmitToken(permission.ConvertToJWT(token))
	code1 = cs.SubmitToken(permission.ConvertToJWT(tokenWrongAudience))

	ctx, cancel = context.WithCancel(context.Background())

	go s0.Dial(ctx, audience+session+"?code="+code0)
	go s1.Dial(ctx, audience+session+"?code="+code1) //connects to correct audience

	time.Sleep(timeout)

	data = []byte("nohopeofgettingthere")
	s0.Out <- reconws.WsMessage{Data: data, Type: websocket.TextMessage}
	select {
	case <-s1.In:
		t.Fatal("TestEnforceAudience...FAIL")
	case <-time.After(timeout):
		t.Log("TestEnforceAudience...PASS")
	}

	cancel()
	time.Sleep(timeout)

	// *** TestEnforceSessionID

	// reader connects with a token intended for a different session on same server
	// should NOT receive message!
	scopes = []string{"read"}
	tokenWrongSessionID := MakeTestToken(audience, "/session/wrongone", scopes, 5)

	code0 = cs.SubmitToken(permission.ConvertToJWT(token))
	code1 = cs.SubmitToken(permission.ConvertToJWT(tokenWrongSessionID))

	ctx, cancel = context.WithCancel(context.Background())

	go s0.Dial(ctx, audience+session+"?code="+code0)
	go s1.Dial(ctx, audience+session+"?code="+code1) //connects to correct session

	data = []byte("notgoingtogetthis")
	s0.Out <- reconws.WsMessage{Data: data, Type: websocket.TextMessage}
	select {
	case <-s1.In:
		t.Fatal("TestEnforceSessionID...FAIL")
	case <-time.After(timeout):
		t.Log("TestEnforceSessionID...PASS")
	}

	cancel()
	time.Sleep(timeout)

	// *** TestEnforceExpiresAt

	// reader connects, sends messages, then is disconnected when session expires
	session = "/session/20fd9a71-2248-4f60-89e3-5d5bb2e78e09"
	scopes = []string{"read", "write"}
	tokenLong := MakeTestToken(audience, session, scopes, 5)
	tokenShort := MakeTestToken(audience, session, scopes, 2)

	code0 = cs.SubmitToken(permission.ConvertToJWT(tokenLong))
	code1 = cs.SubmitToken(permission.ConvertToJWT(tokenShort))

	ctx, cancel = context.WithCancel(context.Background())

	go s0.Dial(ctx, audience+session+"?code="+code0)
	go s1.Dial(ctx, audience+session+"?code="+code1)

	time.Sleep(timeout)

	data = []byte("wanttohearsomething?")
	s0.Out <- reconws.WsMessage{Data: data, Type: websocket.TextMessage}
	select {
	case msg := <-s1.In:
		assert.Equal(t, data, msg.Data)
	case <-time.After(timeout):
		t.Fatal("TestEnforceExpiresAt...FAIL")
	}

	// wait for s1's session to expire
	time.Sleep(2 * time.Second)

	data = []byte("wohtooslow")
	s0.Out <- reconws.WsMessage{Data: data, Type: websocket.TextMessage}
	select {
	case <-s1.In:
		t.Fatal("TestEnforceExpiresAt...FAIL")
	case <-time.After(timeout):
		t.Log("TestEnforceExpiresAt...PASS")
	}
	cancel()
	time.Sleep(timeout)

	// Teardown crossbar
	time.Sleep(timeout)
	close(closed)
	wg.Wait()

}
