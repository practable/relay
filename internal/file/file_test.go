package file

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
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
	return err == nil
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
	config := relay.Config{
		AccessPort:       accessPort,
		RelayPort:        relayPort,
		Audience:         audience,
		Secret:           secret,
		Target:           target,
		AllowNoBookingID: true,
		PruneEvery:       time.Duration(time.Minute),
	}
	go func() {
		// We can't start, stop and restart the relay.Relay without causing mux issues due to net/http
		// It panics on registration of multiple handlers
		// in this case it seems ok with starting straight away
		go relay.Relay(closed, &wg, config)
	}()

	// Sign and get the complete encoded token as a string using the secret
	bearer, err := makeTestToken(audience, secret, 60)

	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	s0 := reconws.New()
	go s0.ReconnectAuth(ctx, audience+"/session/123", bearer)

	select {
	case <-s0.Connected:
		// connected ok
	case <-time.After(2 * time.Second):
		t.Fatalf("s0 failed to connect to relay")
	}

	sighup := make(chan os.Signal)

	testlog := "./test/test.log"
	testlog1 := "./test/test.log.1"

	if exists(testlog) {
		err = os.Remove(testlog)
		assert.NoError(t, err)
	}
	if exists(testlog1) {
		err = os.Remove(testlog1)
		assert.NoError(t, err)
	}

	connected := make(chan struct{})

	// run the file logger
	go func() {
		err = Run(ctx, sighup, connected, audience+"/session/123", bearer, testlog, false)
		assert.NoError(t, err)
	}()

	// we're waiting for the logger to start and connect

	select {
	case <-connected:
		// connected ok
	case <-time.After(2 * time.Second):
		t.Fatalf("file logger failed to connect to relay")
	}

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

	expt := []string{`This is the zeroth message`,
		`This is the first message`,
		`This is the second message`,
		`This is the third message`,
		`This is blank message to prevent index overrun`,
	}

	expectedCount := 4
	actual := bufio.NewScanner(strings.NewReader(s))

	idx := 0
	re := regexp.MustCompile(`^\s*\[[^\]]+\]\s*(.*)`)
	for actual.Scan() {
		parsed := re.FindStringSubmatch(actual.Text())
		assert.Equal(t, 2, len(parsed), "result checking regexp not working correctly, check test code")
		if !(parsed[1] == expt[idx] || parsed[1] == expt[idx+1]) {
			t.Errorf("text does not match;\n got: %s\nexp: %s or %s\n", parsed[1], expt[idx], expt[idx+1])
		}
		idx++
		assert.GreaterOrEqual(t, expectedCount, idx, "too many lines in file")
	}

	// expect at least three messages
	assert.LessOrEqual(t, expectedCount-1, idx)

	if exists(testlog) {
		err = os.Remove(testlog)
		assert.NoError(t, err)
	}

	// playfile test was removed from here, with a cancel and a sleep

	t.Logf("starting sighup test")

	// test the sighup
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	if exists(testlog) {
		err = os.Remove(testlog)
		assert.NoError(t, err)
	}

	t.Logf("starting s0 reconws")
	s0 = reconws.New()
	go s0.ReconnectAuth(ctx, audience+"/session/123", bearer)

	connected = make(chan struct{}) //renew connected channel since closed on previous test

	go func() {
		err = Run(ctx, sighup, connected, audience+"/session/123", bearer, testlog, false)
		assert.NoError(t, err)
	}()

	go func() {
		idx := 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(100 * time.Millisecond)
				s0.Out <- reconws.WsMessage{
					Type: websocket.TextMessage,
					Data: []byte(fmt.Sprintf("Message %d", idx)),
				}
				idx++
			}
		}
	}()

	time.Sleep(time.Second)

	t.Logf("renaming logfile")
	err = os.Rename(testlog, testlog1)
	assert.NoError(t, err)
	t.Logf("issuing sighup")
	sighup <- syscall.SIGHUP
	t.Logf("sighup issued")
	time.Sleep(time.Second)

	cancel()
	t.Logf("reading log files")
	newf, err := os.ReadFile(testlog)
	assert.NoError(t, err)
	news := string(newf)
	t.Logf(news)

	newa := bufio.NewScanner(strings.NewReader(news))

	newCount := 0
	for newa.Scan() {
		newCount++
	}

	oldf, err := os.ReadFile(testlog)
	assert.NoError(t, err)
	olds := string(oldf)
	t.Logf(olds)

	olda := bufio.NewScanner(strings.NewReader(olds))

	oldCount := 0
	for olda.Scan() {
		oldCount++
	}

	// check there are at least three lines in each of the files
	// allowing some room for the exact count to vary from
	// run to run due to external timings we cannot control
	assert.Less(t, 3, newCount)
	assert.Less(t, 3, oldCount)

	// TODO TEST binary mode

	//t.Error("No binary mode test implemented yet")

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
