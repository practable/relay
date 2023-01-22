package relay

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/websocket"
	"github.com/phayes/freeport"
	"github.com/practable/relay/internal/access/restapi/operations"
	"github.com/practable/relay/internal/permission"
	"github.com/practable/relay/internal/reconws"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestRelay(t *testing.T) {

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

	fmt.Printf("audience:%s\n", audience)
	fmt.Printf("target:%s\n", target)

	secret := "testsecret"

	wg.Add(1)

	config := Config{
		AccessPort:       accessPort,
		RelayPort:        relayPort,
		Audience:         audience,
		Secret:           secret,
		Target:           target,
		AllowNoBookingID: true,
		PruneEvery:       time.Duration(time.Minute),
	}

	go Relay(closed, &wg, config) //accessPort, relayPort, audience, secret, target)

	time.Sleep(time.Second) // big safety margin to get crossbar running

	// Start tests

	// TestBidirectionalChat

	client := &http.Client{}

	var claims permission.Token

	start := jwt.NewNumericDate(time.Now().Add(-time.Second))
	after5 := jwt.NewNumericDate(time.Now().Add(5 * time.Second))
	claims.IssuedAt = start
	claims.NotBefore = start
	claims.ExpiresAt = after5

	claims.Audience = jwt.ClaimStrings{audience}
	claims.Topic = "123"
	claims.ConnectionType = "session"
	claims.Scopes = []string{"read", "write"}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign and get the complete encoded token as a string using the secret
	bearer, err := token.SignedString([]byte(secret))
	assert.NoError(t, err)

	// clientPing gets uri with code
	req, err := http.NewRequest("POST", audience+"/session/123", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	body, _ := ioutil.ReadAll(resp.Body)

	var ping operations.SessionOKBody
	err = json.Unmarshal(body, &ping)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(ping.URI, target+"/session/123?code="))

	// clientPong gets uri with code
	req, err = http.NewRequest("POST", audience+"/session/123", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)

	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, _ = ioutil.ReadAll(resp.Body)

	var pong operations.SessionOKBody
	err = json.Unmarshal(body, &pong)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(pong.URI, target+"/session/123?code="))

	// now clients connect using their uris...

	var timeout = 100 * time.Millisecond
	time.Sleep(timeout)

	ctx, cancel := context.WithCancel(context.Background())

	s0 := reconws.New()
	go func() {
		err := s0.Dial(ctx, ping.URI)
		assert.NoError(t, err)
	}()

	s1 := reconws.New()

	go func() {
		err := s1.Dial(ctx, pong.URI)
		assert.NoError(t, err)
	}()

	time.Sleep(timeout)

	data := []byte("ping")

	s0.Out <- reconws.WsMessage{Data: data, Type: websocket.TextMessage}

	select {
	case msg := <-s1.In:
		assert.Equal(t, data, msg.Data)
	case <-time.After(timeout):
		cancel()
		t.Fatal("TestBidirectionalChat...FAIL")
	}

	data = []byte("pong")

	s1.Out <- reconws.WsMessage{Data: data, Type: websocket.TextMessage}

	select {
	case msg := <-s0.In:
		assert.Equal(t, data, msg.Data)
		t.Logf("TestBidirectionalChat...PASS\n")
	case <-time.After(timeout):
		t.Fatal("TestBidirectinalChat...FAIL")
	}
	cancel()

	// TestPreventValidCodeAtWrongSessionID

	// reuse client, ping, pong, token etc from previous test

	// clientPing gets uri with code
	req, err = http.NewRequest("POST", audience+"/session/123", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)

	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, _ = ioutil.ReadAll(resp.Body)

	err = json.Unmarshal(body, &ping)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(ping.URI, target+"/session/123?code="))

	// clientPong gets uri with code
	req, err = http.NewRequest("POST", audience+"/session/123", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)

	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, _ = ioutil.ReadAll(resp.Body)

	err = json.Unmarshal(body, &pong)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(pong.URI, target+"/session/123?code="))

	// now clients connect using their uris...

	time.Sleep(timeout)

	ctx, cancel = context.WithCancel(context.Background())

	go func() {
		err := s0.Dial(ctx, strings.Replace(ping.URI, "123", "456", 1))
		assert.NoError(t, err)
	}()

	go func() {
		err := s1.Dial(ctx, strings.Replace(pong.URI, "123", "456", 1))
		assert.NoError(t, err)
	}()

	time.Sleep(timeout)

	data = []byte("ping")

	s0.Out <- reconws.WsMessage{Data: data, Type: websocket.TextMessage}

	select {
	case msg := <-s1.In:
		t.Fatal("TestPreventValidCodeAtWrongSessionID...FAIL")
		assert.Equal(t, data, msg.Data)
	case <-time.After(timeout):
		cancel()
		t.Logf("TestPreventValidCodeAtWrongSessionID...PASS")
	}
	cancel()

	// Make a connection with a BID in token
	start = jwt.NewNumericDate(time.Now().Add(-time.Second))
	after5 = jwt.NewNumericDate(time.Now().Add(5 * time.Second))
	claims.IssuedAt = start
	claims.NotBefore = start
	claims.ExpiresAt = after5
	claims.BookingID = "bid0"
	token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	bearer, err = token.SignedString([]byte(secret))
	assert.NoError(t, err)

	req, err = http.NewRequest("POST", audience+"/session/123", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)

	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, _ = ioutil.ReadAll(resp.Body)

	if debug {
		t.Log(string(resp.Raw()))
	}

	err = json.Unmarshal(body, &ping)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(ping.URI, target+"/session/123?code="))

	time.Sleep(timeout)

	ctx, cancel = context.WithCancel(context.Background())

	s0 = reconws.New()
	finished := make(chan error)
	testdone := make(chan struct{})

	go func() {
		finished <- errors.New("starting")
		err := s0.Dial(ctx, ping.URI)
		finished <- err
		assert.NoError(t, err)
	}()

	go func() {
	STARTING:
		for {
			select {
			case err := <-finished:
				if err.Error() == "starting" {
					break STARTING
				}
			case <-time.After(5 * time.Second):
				t.Log("Too slow to connect")
				t.Error("TestDenyClosesCurrentConnection...FAIL")
				break STARTING
			}
		}

	DONE:
		for {
			select {
			case err := <-finished:
				if err == nil {
					t.Logf("TestDenyClosesCurrentConnection...PASS")
					break DONE
				} else {
					t.Logf(err.Error())
					t.Logf("TestDenyClosesCurrentConnection...FAIL")
				}
			case <-time.After(time.Second):
				t.Error("TestDenyClosesCurrentConnection...FAIL")
				break DONE
			}
		}
		close(testdone)
	}()

	// Cancel the BID, connection must drop (as checked by the closing of the finish channel
	time.Sleep(time.Millisecond)

	start = jwt.NewNumericDate(time.Now().Add(-time.Second))
	after5 = jwt.NewNumericDate(time.Now().Add(5 * time.Second))
	var adminClaims permission.Token
	adminClaims.Audience = jwt.ClaimStrings{audience}
	adminClaims.IssuedAt = start
	adminClaims.NotBefore = start
	adminClaims.ExpiresAt = after5
	adminClaims.Scopes = []string{"relay:admin"}
	adminToken := jwt.NewWithClaims(jwt.SigningMethodHS256, adminClaims)
	adminBearer, err := adminToken.SignedString([]byte(secret))
	assert.NoError(t, err)

	// POST a deny request
	req, err = http.NewRequest("POST", audience+"/bids/deny", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	q := req.URL.Query()
	q.Add("bid", "bid0")
	q.Add("exp", strconv.Itoa(int(time.Now().Unix()+5)))
	req.URL.RawQuery = q.Encode()
	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)

	<-testdone

	cancel()

	// Try to remake a connection, must fail
	start = jwt.NewNumericDate(time.Now().Add(-time.Second))
	after5 = jwt.NewNumericDate(time.Now().Add(5 * time.Second))
	claims.IssuedAt = start
	claims.NotBefore = start
	claims.ExpiresAt = after5
	claims.BookingID = "bid0"
	token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	bearer, err = token.SignedString([]byte(secret))
	assert.NoError(t, err)

	req, err = http.NewRequest("POST", audience+"/session/123", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, _ = ioutil.ReadAll(resp.Body)
	bodyStr := string([]byte(body))
	expStr := `{"code":"400","message":"bookingID has been deny-listed, probably because the session was cancelled"}` + "\n"

	if expStr == bodyStr {
		t.Logf("TestDeniedBookingIDCannotConnect...PASS")
	} else {
		assert.Equal(t, expStr, bodyStr)
	}

	// teardown relay

	close(closed)
	wg.Wait()

}
