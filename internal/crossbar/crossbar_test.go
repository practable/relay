package crossbar

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/phayes/freeport"
	"github.com/practable/relay/internal/deny"
	"github.com/practable/relay/internal/permission"
	"github.com/practable/relay/internal/reconws"
	"github.com/practable/relay/internal/ttlcode"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// NOTE don't use reconws.Reconnect for production clients anymore;
// it does NOT understand the use of auth codes
// use Dial instead.

func MakeTestToken(audience, connectionType, topic string, scopes []string, lifetime int64) permission.Token {
	begin := time.Now().Unix() - 1 //ensure it's in the past
	end := begin + lifetime
	return permission.NewToken(audience, connectionType, topic, scopes, begin, begin, end)
}

func TestCrossbar(t *testing.T) {

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
	denied := make(chan string)
	var wg sync.WaitGroup

	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	audience := "ws://127.0.0.1:" + strconv.Itoa(port)
	secret := "somesecret"
	cs := ttlcode.NewDefaultCodeStore()
	ds := deny.New()

	config := Config{
		Listen:     port,
		Audience:   audience,
		CodeStore:  cs,
		DenyStore:  ds,
		Secret:     secret,
		StatsEvery: time.Duration(time.Second),
	}

	wg.Add(1)
	go Crossbar(config, closed, denied, &wg)
	// safety margin to get crossbar running
	time.Sleep(time.Second)

	var timeout = 100 * time.Millisecond

	// Start tests

	// *** TestCanConnectWithValidCode ***
	// these parameters reused by following tests:

	ctx, cancel := context.WithCancel(context.Background())

	ct := "session"
	session := "20fd9a71-2248-4f60-89e3-5d5bb2e78e09"
	scopes := []string{"read", "write"}

	token := MakeTestToken(audience, ct, session, scopes, 5)

	code0 := cs.SubmitToken(token)
	code1 := cs.SubmitToken(token)

	s0 := reconws.New()
	go func() {
		err := s0.Dial(ctx, audience+"/"+ct+"/"+session+"?code="+code0)
		assert.NoError(t, err)
	}()

	s1 := reconws.New()

	go func() {
		err := s1.Dial(ctx, audience+"/"+ct+"/"+session+"?code="+code1)
		assert.NoError(t, err)
	}()

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

	// while connected, get stats
	statsToken := MakeTestToken(audience, ct, "stats", scopes, 5)
	statsCode := cs.SubmitToken(statsToken)
	stats := reconws.New()
	go func() {
		err := stats.Dial(ctx, audience+"/"+ct+"/stats?code="+statsCode)
		assert.NoError(t, err)
	}()

	//TODO send message, receive stats

	cmd, err := json.Marshal(StatsCommand{Command: "update"})

	assert.NoError(t, err)

	stats.Out <- reconws.WsMessage{Data: cmd, Type: websocket.TextMessage}

	select {
	case msg := <-stats.In:

		t.Log("TestGetStats...PROVISIONAL-PASS")

		var reports []*ClientReport

		err := json.Unmarshal(msg.Data, &reports)

		assert.NoError(t, err)

		agents := make(map[string]int)

		for _, report := range reports {
			count, ok := agents[report.Topic]
			if !ok {
				agents[report.Topic] = 1
				continue
			}

			agents[report.Topic] = count + 1
		}

		if agents[session] == 2 {
			t.Log("TestGetStats...PASS")
		} else {
			t.Fatalf("TestGetStats...FAIL")
		}

	case <-time.After(timeout):
		t.Fatalf("TestGetStats...FAIL")
	}

	cancel()
	time.Sleep(timeout)

	// *** TestCannotConnectWithReusedCode ***
	// try the last test again, without getting new codes

	ctx, cancel = context.WithCancel(context.Background())

	go func() {
		err := s0.Dial(ctx, audience+"/"+ct+"/"+session+"?code="+code0)
		assert.NoError(t, err)
	}()

	go func() {
		err := s1.Dial(ctx, audience+"/"+ct+"/"+session+"?code="+code1)
		assert.NoError(t, err)
	}()

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

	code0 = cs.SubmitToken(token)
	code1 = cs.SubmitToken(token)

	ctx, cancel = context.WithCancel(context.Background())

	go func() {
		err := s0.Dial(ctx, audience+"/"+ct+"/"+session+"?code="+code0)
		assert.NoError(t, err)
	}()

	go func() {
		err := s1.Dial(ctx, audience+"/"+ct+"/"+session+"?code="+code1)
		assert.NoError(t, err)
	}()

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
	tokenReadOnly := MakeTestToken(audience, ct, session, scopes, 5)

	code0 = cs.SubmitToken(token)
	code1 = cs.SubmitToken(tokenReadOnly)

	ctx, cancel = context.WithCancel(context.Background())

	go func() {
		err := s0.Dial(ctx, audience+"/"+ct+"/"+session+"?code="+code0)
		assert.NoError(t, err)
	}()

	go func() {
		err := s1.Dial(ctx, audience+"/"+ct+"/"+session+"?code="+code1)
		assert.NoError(t, err)
	}()

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
	tokenWrongAudience := MakeTestToken("ws://wrong.server.io", ct, session, scopes, 5)

	code0 = cs.SubmitToken(token)
	code1 = cs.SubmitToken(tokenWrongAudience)

	ctx, cancel = context.WithCancel(context.Background())

	go func() {
		err := s0.Dial(ctx, audience+"/"+ct+"/"+session+"?code="+code0)
		assert.NoError(t, err)
	}()

	go func() {
		err := s1.Dial(ctx, audience+"/"+ct+"/"+session+"?code="+code1)
		assert.NoError(t, err)
	}()

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
	code0 = cs.SubmitToken(token)
	code1 = cs.SubmitToken(token)

	ctx, cancel = context.WithCancel(context.Background())

	go func() {
		err := s0.Dial(ctx, audience+"/"+ct+"/notMySession?code="+code0)
		assert.NoError(t, err)
	}()

	go func() {
		err := s1.Dial(ctx, audience+"/"+ct+"/notMySession?code="+code1)
		assert.NoError(t, err)
	}()

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
	session = "20fd9a71-2248-4f60-89e3-5d5bb2e78e09"
	scopes = []string{"read", "write"}
	tokenLong := MakeTestToken(audience, ct, session, scopes, 5)
	tokenShort := MakeTestToken(audience, ct, session, scopes, 2)

	code0 = cs.SubmitToken(tokenLong)
	code1 = cs.SubmitToken(tokenShort)

	ctx, cancel = context.WithCancel(context.Background())

	go func() {
		err := s0.Dial(ctx, audience+"/"+ct+"/"+session+"?code="+code0)
		assert.NoError(t, err)
	}()

	go func() {
		err := s1.Dial(ctx, audience+"/"+ct+"/"+session+"?code="+code1)
		assert.NoError(t, err)
	}()

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

func BenchmarkSmallMessage(b *testing.B) {

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
	denied := make(chan string)
	var wg sync.WaitGroup

	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	audience := "ws://127.0.0.1:" + strconv.Itoa(port)
	cs := ttlcode.NewDefaultCodeStore()
	ds := deny.New()

	config := Config{
		Listen:    port,
		Audience:  audience,
		CodeStore: cs,
		DenyStore: ds,
	}

	wg.Add(1)
	go Crossbar(config, closed, denied, &wg)

	var timeout = 5 * time.Millisecond

	// safety margin to get crossbar running
	time.Sleep(timeout)

	// Start tests

	// *** TestCanConnectWithValidCode ***
	// these parameters reused by:

	ctx, cancel := context.WithCancel(context.Background())
	ct := "session"
	session := "session/20fd9a71-2248-4f60-89e3-5d5bb2e78e09"
	scopes := []string{"read", "write"}

	token := MakeTestToken(audience, ct, session, scopes, 5)

	code0 := cs.SubmitToken(token)
	code1 := cs.SubmitToken(token)

	s0 := reconws.New()
	go func() {
		err := s0.Dial(ctx, audience+"/"+ct+"/"+session+"?code="+code0)
		assert.NoError(b, err)
	}()

	s1 := reconws.New()

	go func() {
		err := s1.Dial(ctx, audience+"/"+ct+"/"+session+"?code="+code1)
		assert.NoError(b, err)
	}()

	time.Sleep(timeout)

	data := []byte("foo")

	msgOut := reconws.WsMessage{Data: data, Type: websocket.TextMessage}

	var msg reconws.WsMessage

	for n := 0; n < b.N; n++ {
		for m := 0; m < 1000; m++ {
			s0.Out <- msgOut
			// always record the result to prevent
			// the compiler eliminating the function call.
			msg = <-s1.In
		}
	}

	// always store the result to a package level variable
	// so the compiler cannot eliminate the Benchmark itself.
	assert.Equal(b, data, msg.Data)

	cancel()
	time.Sleep(timeout)

	// Teardown crossbar
	time.Sleep(timeout)
	close(closed)
	wg.Wait()

}

func BenchmarkLargeMessage(b *testing.B) {

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
	denied := make(chan string)
	var wg sync.WaitGroup

	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	audience := "ws://127.0.0.1:" + strconv.Itoa(port)
	cs := ttlcode.NewDefaultCodeStore()
	ds := deny.New()

	config := Config{
		Listen:    port,
		Audience:  audience,
		CodeStore: cs,
		DenyStore: ds,
	}

	wg.Add(1)
	go Crossbar(config, closed, denied, &wg)

	var timeout = 5 * time.Millisecond

	// safety margin to get crossbar running
	time.Sleep(timeout)

	// Start tests

	// *** TestCanConnectWithValidCode ***
	// these parameters reused by:

	ctx, cancel := context.WithCancel(context.Background())
	ct := "session"
	session := "20fd9a71-2248-4f60-89e3-5d5bb2e78e09"
	scopes := []string{"read", "write"}

	token := MakeTestToken(audience, ct, session, scopes, 5)

	code0 := cs.SubmitToken(token)
	code1 := cs.SubmitToken(token)

	s0 := reconws.New()
	go func() {
		err := s0.Dial(ctx, audience+"/"+ct+"/"+session+"?code="+code0)
		assert.NoError(b, err)
	}()

	s1 := reconws.New()

	go func() {
		err := s1.Dial(ctx, audience+"/"+ct+"/"+session+"?code="+code1)
		assert.NoError(b, err)
	}()

	time.Sleep(timeout)

	data := make([]byte, 1024*1024)

	_, err = rand.Read(data)
	assert.NoError(b, err)

	msgOut := reconws.WsMessage{Data: data, Type: websocket.TextMessage}

	var msg reconws.WsMessage

	for n := 0; n < b.N; n++ {
		for m := 0; m < 100; m++ {
			s0.Out <- msgOut
			// always record the result to prevent
			// the compiler eliminating the function call.
			msg = <-s1.In
		}
	}

	// always store the result to a package level variable
	// so the compiler cannot eliminate the Benchmark itself.
	assert.Equal(b, data, msg.Data)

	cancel()
	time.Sleep(timeout)

	// Teardown crossbar
	time.Sleep(timeout)
	close(closed)
	wg.Wait()

}

func BenchmarkLargeRandomPacketGeneration(b *testing.B) {

	data := make([]byte, 1024*1024)
	var ping, pong reconws.WsMessage
	bar := make(chan reconws.WsMessage, 2)
	for n := 0; n < b.N; n++ {
		for m := 0; m < 100; m++ {
			_, err := rand.Read(data)
			assert.NoError(b, err)
			ping = reconws.WsMessage{Data: data, Type: websocket.TextMessage}
			bar <- ping
			pong = <-bar
		}
	}
	assert.Equal(b, pong.Data, data)
}

func BenchmarkLargeRandomMessage(b *testing.B) {

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
	denied := make(chan string)
	var wg sync.WaitGroup

	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	audience := "ws://127.0.0.1:" + strconv.Itoa(port)
	cs := ttlcode.NewDefaultCodeStore()
	ds := deny.New()

	config := Config{
		Listen:    port,
		Audience:  audience,
		CodeStore: cs,
		DenyStore: ds,
	}

	wg.Add(1)
	go Crossbar(config, closed, denied, &wg)

	var timeout = 5 * time.Millisecond

	// safety margin to get crossbar running
	time.Sleep(timeout)

	// Start tests

	// *** TestCanConnectWithValidCode ***
	// these parameters reused by:

	ctx, cancel := context.WithCancel(context.Background())
	ct := "session"
	session := "session/20fd9a71-2248-4f60-89e3-5d5bb2e78e09"
	scopes := []string{"read", "write"}

	token := MakeTestToken(audience, ct, session, scopes, 5)

	code0 := cs.SubmitToken(token)
	code1 := cs.SubmitToken(token)

	s0 := reconws.New()
	go func() {
		err := s0.Dial(ctx, audience+"/"+ct+"/"+session+"?code="+code0)
		assert.NoError(b, err)
	}()

	s1 := reconws.New()

	go func() {
		err := s1.Dial(ctx, audience+"/"+ct+"/"+session+"?code="+code1)
		assert.NoError(b, err)
	}()

	time.Sleep(timeout)

	data := make([]byte, 1024*1024)

	var msg reconws.WsMessage

	for n := 0; n < b.N; n++ {
		for m := 0; m < 100; m++ {
			_, err := rand.Read(data)
			assert.NoError(b, err)
			s0.Out <- reconws.WsMessage{Data: data, Type: websocket.TextMessage}
			// always record the result to prevent
			// the compiler eliminating the function call.
			msg = <-s1.In
		}
	}

	// always store the result to a package level variable
	// so the compiler cannot eliminate the Benchmark itself.
	assert.Equal(b, data, msg.Data)

	cancel()
	time.Sleep(timeout)

	// Teardown crossbar
	time.Sleep(timeout)
	close(closed)
	wg.Wait()

}

func TestSlashify(t *testing.T) {

	if "/foo" != slashify("foo") {
		t.Errorf("Slashify not prefixing slash ")
	}
	if "//foo" == slashify("/foo") {
		t.Errorf("Slashify prefixing additional slash")
	}
	if "/foo" != slashify("/foo/") {
		t.Errorf("Slashify not removing trailing slash")
	}
	if "/foo" != slashify("foo/") {
		t.Errorf("Slashify not both removing trailing slash AND prefixing slash")
	}

	b := "foo/bar/rab/oof/"
	if "/foo/bar/rab/oof" != slashify(b) {
		t.Errorf("Slashify not coping with internal slashes %s -> %s", b, slashify(b))
	}

}

func TestGetConnectionTypeFromPath(t *testing.T) {

	assert.Equal(t, "connectionType", getConnectionTypeFromPath("/connectionType/sessionID"))
	assert.Equal(t, "", getConnectionTypeFromPath("NoLeadingSlash/A/B/C"))
	assert.Equal(t, "foo%20bar", getConnectionTypeFromPath("/foo%20bar/glum"))
	assert.Equal(t, "", getConnectionTypeFromPath("ooops/foo%20bar/glum"))

}
func TestGetTopicFromPath(t *testing.T) {

	assert.Equal(t, "sessionID", getTopicFromPath("/connectionType/sessionID"))
	assert.Equal(t, "", getTopicFromPath("NoLeadingSlash/A/B/C"))
	assert.Equal(t, "session%20ID/connection%20ID", getTopicFromPath("/connectionType/session%20ID/connection%20ID"))
	assert.Equal(t, "sessionID/connectionID", getTopicFromPath("/connectionType/sessionID/connectionID?QueryParams=Something"))
	assert.Equal(t, "sessionID/connectionID", getTopicFromPath("/connectionType/sessionID/connectionID?QueryParams=Something&SomeThing=Else"))
}

func TestGetSessionIDFromPath(t *testing.T) {

	assert.Equal(t, "sessionID", getSessionIDFromPath("/connectionType/sessionID"))
	assert.Equal(t, "", getSessionIDFromPath("NoLeadingSlash/A/B/C"))
	assert.Equal(t, "session%20ID", getSessionIDFromPath("/connectionType/session%20ID/connection%20ID"))
	assert.Equal(t, "sessionID", getSessionIDFromPath("/connectionType/sessionID/connectionID?QueryParams=Something"))
	assert.Equal(t, "sessionID", getSessionIDFromPath("/connectionType/sessionID/connectionID?QueryParams=Something&SomeThing=Else"))
}

func TestGetConnectionIDFromPath(t *testing.T) {

	assert.Equal(t, "", getConnectionIDFromPath("/connectionType/sessionID"))
	assert.Equal(t, "", getConnectionIDFromPath("NoLeadingSlash/A/B/C"))
	assert.Equal(t, "connection%20ID", getConnectionIDFromPath("/connectionType/session%20ID/connection%20ID  "))
	assert.Equal(t, "connectionID", getConnectionIDFromPath("/connectionType/sessionID/connectionID?QueryParams=Something"))
	assert.Equal(t, "connectionID", getConnectionIDFromPath("/connectionType/sessionID/connectionID?QueryParams=Something&SomeThing=Else"))

}
