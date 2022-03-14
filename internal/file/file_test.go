package file

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/websocket"
	"github.com/phayes/freeport"
	"github.com/practable/relay/internal/permission"
	"github.com/practable/relay/internal/reconws"
	"github.com/practable/relay/internal/relay"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	flag.Parse() // needed to use testing.Verbose https://github.com/golang/go/issues/9825

	if testing.Verbose() {
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true, DisableColors: true})
		defer log.SetOutput(os.Stdout)

	} else {
		var ignore bytes.Buffer
		logignore := bufio.NewWriter(&ignore)
		log.SetOutput(logignore)
	}

	exitVal := m.Run()

	os.Exit(exitVal)
}

// just for test use, see better approaches here: https://stackoverflow.com/questions/12518876/how-to-check-if-a-file-exists-in-go
func exists(name string) bool {
	_, err := os.Stat(name)
	if err == nil {
		return true
	}
	return false
}

func TestRun(t *testing.T) {

	interval := 10 * time.Millisecond

	// Setup logging
	debug := true

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

	//var timeout = 100 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())

	s0 := reconws.New()
	go s0.ReconnectAuth(ctx, audience+"/session/123", bearer)

	sighup := make(chan os.Signal)

	testlog := "./test/test.log"
	testlog1 := "./test/test.log.1"
	playfilename := "./test/test.play"

	if exists(testlog) {
		err = os.Remove(testlog)
		assert.NoError(t, err)
	}
	if exists(testlog1) {
		err = os.Remove(testlog1)
		assert.NoError(t, err)
	}

	// no play file for now
	go func() {
		err = Run(ctx, sighup, audience+"/session/123", bearer, testlog, "", interval)
		assert.NoError(t, err)
	}()

	time.Sleep(time.Millisecond * 1000)

	s0.Out <- reconws.WsMessage{Type: websocket.TextMessage,
		Data: []byte("This is the zeroth message")}

	time.Sleep(time.Millisecond * 10)

	s0.Out <- reconws.WsMessage{Type: websocket.TextMessage,
		Data: []byte("This is the first message")}

	time.Sleep(time.Millisecond * 10)

	s0.Out <- reconws.WsMessage{Type: websocket.TextMessage,
		Data: []byte("This is the second message")}

	time.Sleep(time.Millisecond * 10)

	s0.Out <- reconws.WsMessage{Type: websocket.TextMessage,
		Data: []byte("This is the third message")}

	time.Sleep(time.Millisecond * 10)

	cancel()

	dat, err := os.ReadFile(testlog)
	assert.NoError(t, err)
	s := string(dat)
	t.Logf(s)

	// let's see if we can get at least two messages
	assert.Less(t, 2, strings.Count(s, "["))

	ec := `This is the zeroth message
This is the first message
This is the second message
This is the third message
`

	expectedCount := 4
	actual := bufio.NewScanner(strings.NewReader(s))
	expected := bufio.NewScanner(strings.NewReader(ec))

	idx := 0
	re := regexp.MustCompile(`^\s*\[[^\]]+\]\s*(.*)`)
	for actual.Scan() {
		expected.Scan() //protected from overrun by final assert in this loop
		parsed := re.FindStringSubmatch(actual.Text())
		assert.Equal(t, 2, len(parsed), "result checking regexp not working correctly, check test code")
		assert.Equal(t, expected.Text(), parsed[1], "text does not match")
		idx++
		assert.GreaterOrEqual(t, expectedCount, idx, "too many lines in file")
	}

	if exists(testlog) {
		err = os.Remove(testlog)
		assert.NoError(t, err)
	}

	// now try playing a file
	ctx, cancel = context.WithCancel(context.Background())

	play := `{"some":"msg"}
# Non echo comment
#- non echo comment
#+ echo comment
[0.1s] {"an":"other"}
[1ms] {"an":"other"}
<'^foo\s*',5,10ms> {"send":"foos"}
[10ms]
[1ms]a
[1ms]b
[1ms]c
[1ms]d
[1ms]e
[1ms]f
[1ms]g
[1ms]
#+ start set filter
|+> [a-h]
|accept> [R-Z]
|->[0-9]
|deny>  [#!&%]
#+ done set filter
[10ms]
[1ms]ah0#
[1ms]AA
[1ms]ZZ
[1ms]abc
[1ms]abc!
[1ms]ah
[10ms]
|reset>
#+ reset
[10ms]
[1ms]ah0#
[1ms]AA
[1ms]ZZ
[1ms]abc
[1ms]abc!
[1ms]ah
[1s]
`
	err = os.WriteFile(playfilename, []byte(play), 0644)
	assert.NoError(t, err)

	// echo to self, see that filter works...
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	if exists(testlog) {
		err = os.Remove(testlog)
		assert.NoError(t, err)
	}

	s0 = reconws.New()
	go s0.ReconnectAuth(ctx, audience+"/session/123", bearer)

	//echo messages back without modification
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-s0.In:
				s0.Out <- msg
			}
		}
	}()

	time.Sleep(10 * time.Millisecond)
	err = Run(ctx, sighup, audience+"/session/123", bearer, testlog, playfilename, interval)
	assert.NoError(t, err)

	dat, err = os.ReadFile(testlog)
	assert.NoError(t, err)
	s = string(dat)
	t.Logf(s)

	ec = `echo comment
{"some":"msg"}
{"an":"other"}
{"an":"other"}
{"send":"foos"}
a
b
c
d
e
f
g
start set filter
done set filter
ZZ
abc
ah
reset
ah0#
AA
ZZ
abc
abc!
ah
` //put ` on this line so last line is processed

	expectedCount = 23
	actual = bufio.NewScanner(strings.NewReader(s))
	expected = bufio.NewScanner(strings.NewReader(ec))

	idx = 0
	re = regexp.MustCompile(`^\s*\[[^\]]+\]\s*(.*)`)
	for actual.Scan() {
		expected.Scan() //protected from overrun by final assert in this loop
		parsed := re.FindStringSubmatch(actual.Text())
		assert.Equal(t, 2, len(parsed))
		assert.Equal(t, expected.Text(), parsed[1], "text does not match")
		idx++
		assert.GreaterOrEqual(t, expectedCount, idx, "too many lines in file")
	}

	assert.Equal(t, expectedCount, idx, "incorrect number of lines in file")

	// Shutdown the Relay and check no messages are being sent
	close(closed)
	wg.Wait()

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
