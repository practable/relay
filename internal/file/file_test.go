package file

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"net/http"
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

func TestParseByLine(t *testing.T) {

	// put closing quote on its own line to ensure test counts lines
	// correctly (else test will hang forever)
	s := `{"some":"msg"}
# Non echo comment
#- non echo comment
#+ echo comment
[0.1s] {"an":"other"}
[] {"an":"other"}
<'^foo\s*',5,0.3h1.5m0.1s> {"send":"foos"}
[0.1] {"an":"other"}
<'^foo\s*',,10s> {"send":"foos"}
<'^foo\s*',5,> {"send":"foos"}
|+> [a-h]
|accept> [R-Z]
|->[0-9]
|deny>  [#!&%]
|reset>
|A> [a-h]
|D> [0-9]
|r> 
|X>
|a> ^\/(?!\/)(.*?)
`

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	go func() {
		select {
		case <-ctx.Done():
		case <-time.After(5 * time.Second):
			// see definition of s (above) and assertion check on n (below)
			fmt.Printf("test hung - check formatting of multiline string")
		}
	}()

	n := strings.Count(s, "\n")
	assert.Equal(t, 20, n) // Update this after editing

	expected := make([]interface{}, n)
	expected[0] = Send{
		Msg: `{"some":"msg"}`,
	}
	expected[1] = Comment{
		Msg:  "Non echo comment",
		Echo: false,
	}

	expected[2] = Comment{
		Msg:  "non echo comment",
		Echo: false,
	}

	expected[3] = Comment{
		Msg:  "echo comment",
		Echo: true,
	}

	expected[4] = Send{
		Msg:   `{"an":"other"}`,
		Delay: time.Millisecond * 100,
	}

	expected[5] = Send{
		Msg: `{"an":"other"}`,
	}

	expected[6] = Send{
		Msg: `{"send":"foos"}`,
		Condition: Condition{
			AcceptPattern: *regexp.MustCompile("^foo\\s*"),
			Count:         5,
			Timeout:       time.Second*((19*60)+30) + time.Millisecond*100,
		},
	}

	expected[7] = Error{`unknown delay time format: [0.1] {"an":"other"}`}

	expected[8] = Error{`malformed condition command [<'^foo\s*',,10s> {"send":"foos"} '^foo\s*',,10s {"send":"foos"}]; second argument  should be integer, count of messages to await. Line was: <'^foo\s*',,10s> {"send":"foos"}`}

	expected[9] = Error{`malformed condition command [<'^foo\s*',5,> {"send":"foos"} '^foo\s*',5, {"send":"foos"}]; third argument  should be timeout duration in format like 10s or 1m. Yours could not be parsed because time: invalid duration "". Line was was <'^foo\s*',5,> {"send":"foos"}`}

	expected[10] = FilterAction{
		Verb:    Accept,
		Pattern: regexp.MustCompile(`[a-h]`),
	}

	expected[11] = FilterAction{
		Verb:    Accept,
		Pattern: regexp.MustCompile(`[R-Z]`),
	}

	expected[12] = FilterAction{
		Verb:    Deny,
		Pattern: regexp.MustCompile(`[0-9]`),
	}

	expected[13] = FilterAction{
		Verb:    Deny,
		Pattern: regexp.MustCompile(`[#!&%]`),
	}

	expected[14] = FilterAction{
		Verb: Reset,
	}

	expected[15] = expected[10]

	expected[16] = expected[12]

	expected[17] = expected[14]

	expected[18] = Error{`malformed filter command; first argument not one of [+,-,a,d,r,accept,deny,reset], but was X`}

	expected[19] = Error{`malformed filter command; last argument ^\/(?!\/)(.*?) should be regexp pattern, but did not compile because error parsing regexp: invalid or unsupported Perl syntax: ` + "`(?!`. " + `Line was |a> ^\/(?!\/)(.*?)`}

	in := strings.NewReader(s)

	out := make(chan interface{}, n) // buffer >= lines in s to avoid hang

	err := ParseByLine(in, out)

	assert.NoError(t, err)

	idx := 0
	for o := range out {
		assert.Equal(t, expected[idx], o)
		t.Logf("%d: %v\n", idx, o)
		idx++
	}

	assert.Equal(t, n, idx)

}

func TestFilter(t *testing.T) {

	p0 := regexp.MustCompile("[a-h]")
	p1 := regexp.MustCompile("[R-Z]")
	p2 := regexp.MustCompile("[0-9]")
	p3 := regexp.MustCompile("[#!&%]")

	f := NewFilter()

	f.AddAcceptPattern(p0)
	f.AddAcceptPattern(p1)
	f.AddDenyPattern(p2)
	f.AddDenyPattern(p3)

	s := `ach
ach0
ach!
TUV23
TUV%
TUV
ACH
tuv
`

	expected := []bool{
		true,
		false,
		false,
		false,
		false,
		true,
		false,
		false,
	}

	n := strings.Count(s, "\n")
	assert.Equal(t, len(expected), n)

	in := strings.NewReader(s)
	scanner := bufio.NewScanner(in)

	idx := 0
	for scanner.Scan() {
		line := scanner.Text()
		assert.Equal(t, expected[idx], f.Pass(line))
		idx++
	}

	// Check all message pass after reset
	f.Reset()
	in = strings.NewReader(s)
	scanner = bufio.NewScanner(in)

	idx = 0
	for scanner.Scan() {
		line := scanner.Text()
		assert.Equal(t, true, f.Pass(line))
		idx++
	}

	// Check whether delete works (have to add two filters
	// to avoid returning to allpass when you delete one)
	assert.Equal(t, 0, len(*f.AcceptPatterns))
	f.AddAcceptPattern(p0)
	f.AddAcceptPattern(p1)
	assert.Equal(t, 2, len(*f.AcceptPatterns))
	assert.True(t, f.Pass("abc"))

	f.DeleteAcceptPattern(p0)
	assert.Equal(t, 1, len(*f.AcceptPatterns))
	assert.False(t, f.Pass("abc"))

	// check deleteDenyPattern
	f.AddDenyPattern(p2)
	f.AddDenyPattern(p3)
	assert.False(t, f.Pass("T!"))
	f.DeleteDenyPattern(p3)
	assert.True(t, f.Pass("T!"))

	// check delete when a different regexp instance is used
	// to avoid implementations that don't compare the operation
	// of filters and instead use something inappropriate like
	// the value of a pointer etc. That would not work when
	// separate lines will each generate their own regexp
	// expressions rather than sharing a pointer
	f.AddDenyPattern(p3)
	assert.False(t, f.Pass("T!"))
	p3a := regexp.MustCompile("[#!&%]")
	f.DeleteDenyPattern(p3a)
	assert.True(t, f.Pass("T!"))

	f.AddAcceptPattern(p0)
	assert.True(t, f.Pass("abc"))
	p0a := regexp.MustCompile("[a-h]")
	f.DeleteAcceptPattern(p0a)
	assert.False(t, f.Pass("abc"))

	// check that only exact filters are deleted
	f.AddAcceptPattern(p0)
	assert.True(t, f.Pass("abc"))
	p5 := regexp.MustCompile("[a-h]\\s*")
	f.DeleteAcceptPattern(p5)
	assert.True(t, f.Pass("abc")) //passed because p0 not deleted

	//regexp.Regexp stores original text, so whitespace IS different
	p0b := regexp.MustCompile(" [a-h]")
	f.DeleteAcceptPattern(p0b)
	assert.True(t, f.Pass("abc")) //passed because p0 not deleted

}

func TestAll(t *testing.T) {

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

	startTime := time.Now().Unix()
	go func() {
		time.Sleep(2 * time.Second)
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

	var timeout = 100 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())

	s0 := reconws.New()
	go s0.ReconnectAuth(ctx, audience+"/session/123", bearer)

	s1 := reconws.New()
	go s1.ReconnectAuth(ctx, audience+"/session/123", bearer)

	time.Sleep(timeout)

	data := []byte("prestart-ping-no-chance")

	select {
	case s0.Out <- reconws.WsMessage{Data: data, Type: websocket.TextMessage}:
		t.Fatal("s0 sent messsage to dead relay")
		select {
		case <-s1.In:
			t.Fatal("no message expected")
		case <-time.After(timeout):
		}
	case <-time.After(timeout):
	}

	data = []byte("prestart-pong-no-chance")

	select { //send may or may not happen
	case s1.Out <- reconws.WsMessage{Data: data, Type: websocket.TextMessage}:
		t.Fatal("s1 sent messsage to dead relay")
		select {
		case <-s0.In:
			t.Fatal("no message expected")
		case <-time.After(timeout):
		}
	case <-time.After(timeout):
	}

	// check we finished test before relay started
	assert.True(t, startTime+2 > time.Now().Unix())

	// now wait until both clients have connected
	// one will connect before the other, so it's not
	// possible to guarantee both get this first message
	// and that is normal behaviour for a non-caching
	// relay....
	data = []byte("hello")
	s0.Out <- reconws.WsMessage{Data: data, Type: websocket.TextMessage}
	s1.Out <- reconws.WsMessage{Data: data, Type: websocket.TextMessage}

	time.Sleep(timeout) // send can come online before receive

	// now send a message we care about
	data0 := []byte("ping")
	s0.Out <- reconws.WsMessage{Data: data0, Type: websocket.TextMessage}
	data1 := []byte("pong")
	s1.Out <- reconws.WsMessage{Data: data1, Type: websocket.TextMessage}

	gotPing := false
	gotPong := false

	for i := 0; i < 20; i++ {
		select {
		case msg := <-s1.In:
			if debug {
				t.Log(string(msg.Data))
			}
			if bytes.Equal(msg.Data, data0) {
				gotPing = true
			}
			// sometimes the messages combine into "helloping"
			// due to the way framing is etsimated in relay
			if bytes.Equal(msg.Data, append(data, data0...)) {
				gotPing = true
			}
		case msg := <-s0.In:
			if debug {
				t.Log(string(msg.Data))
			}
			if bytes.Equal(msg.Data, data1) {
				gotPong = true
				if gotPing {
					break
				}
			}
			if bytes.Equal(msg.Data, append(data, data1...)) {
				gotPing = true
				if gotPong {
					break
				}
			}
		case <-time.After(timeout):
			continue
		}
	}

	if !gotPing || !gotPong {
		t.Error("did not get both messages")
	}

	cancel()
	// Shutdown the Relay and check no messages are being sent
	close(closed)
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
