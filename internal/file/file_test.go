package file

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"os"
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

	go func() {

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

	}()

	// no play file for now
	err = Run(ctx, sighup, audience+"/session/123", bearer, testlog, "")

	assert.NoError(t, err)

	dat, err := os.ReadFile(testlog)
	assert.NoError(t, err)
	s := string(dat)
	t.Logf(s)

	// let's see if we can get at least two messages
	assert.Less(t, 2, strings.Count(s, "["))

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
<'^foo\s*',5,100ms> {"send":"foos"}
|+> [a-h]
|accept> [R-Z]
|->[0-9]
|deny>  [#!&%]
[10ms]ah
[1ms]ah0#
|reset>
[1ms]A1
|A> [a-h]
|D> [0-9]
[1ms]A2
|r> 
[1ms]A3
`
	err = os.WriteFile(playfilename, []byte(play), 0644)

	assert.NoError(t, err)

	go func() {
		time.Sleep(10 * time.Millisecond)
		err = Run(ctx, sighup, audience+"/session/123", bearer, "", playfilename)
		time.Sleep(time.Second)
		cancel()

	}()

	err = Run(ctx, sighup, audience+"/session/123", bearer, testlog, "")
	assert.NoError(t, err)

	dat, err = os.ReadFile(testlog)
	assert.NoError(t, err)
	s = string(dat)
	t.Logf(s)

	//NOTE - there is a condition in the above file which should delay 19minutes ... but isn't!

	// let's see if we can get at least five lines
	assert.Less(t, 2, strings.Count(s, "\n"))

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
	err = Run(ctx, sighup, audience+"/session/123", bearer, testlog, playfilename)

	dat, err = os.ReadFile(testlog)
	assert.NoError(t, err)
	s = string(dat)
	t.Logf(s)

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
