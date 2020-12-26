package relay

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"
	"github.com/phayes/freeport"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/timdrysdale/relay/pkg/access"
	"github.com/timdrysdale/relay/pkg/access/restapi/operations"
	"github.com/timdrysdale/relay/pkg/permission"
	"github.com/timdrysdale/relay/pkg/reconws"
)

func TestRelay(t *testing.T) {

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

	fmt.Printf("audience:%s\n", audience)
	fmt.Printf("target:%s\n", target)

	secret := "testsecret"

	wg.Add(1)

	go Relay(closed, &wg, accessPort, relayPort, audience, secret, target, access.Options{})

	time.Sleep(time.Second) // big safety margin to get crossbar running

	// Start tests

	// TestBidirectionalChat

	client := &http.Client{}

	var claims permission.Token

	claims.IssuedAt = time.Now().Unix() - 1
	claims.NotBefore = time.Now().Unix() - 1
	claims.ExpiresAt = time.Now().Unix() + 5
	claims.Audience = audience
	claims.Topic = "123"
	claims.ConnectionType = "session"
	claims.Scopes = []string{"read", "write"}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign and get the complete encoded token as a string using the secret
	bearer, err := token.SignedString([]byte(secret))
	assert.NoError(t, err)

	// clientPing gets uri with code
	req, err := http.NewRequest("POST", audience+"/session/123", nil)
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
	go s0.Dial(ctx, ping.URI)

	s1 := reconws.New()
	go s1.Dial(ctx, pong.URI)

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
	req.Header.Add("Authorization", bearer)

	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, _ = ioutil.ReadAll(resp.Body)

	err = json.Unmarshal(body, &ping)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(ping.URI, target+"/session/123?code="))

	// clientPong gets uri with code
	req, err = http.NewRequest("POST", audience+"/session/123", nil)
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
	go s0.Dial(ctx, strings.Replace(ping.URI, "123", "456", 1))

	go s1.Dial(ctx, strings.Replace(pong.URI, "123", "456", 1))

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
	// teardown relay

	close(closed)
	wg.Wait()

}
