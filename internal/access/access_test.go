package access

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/phayes/freeport"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/timdrysdale/relay/pkg/access/restapi/operations"
	"github.com/timdrysdale/relay/pkg/permission"
	"github.com/timdrysdale/relay/pkg/ttlcode"
)

func TestGetPrefixFromPath(t *testing.T) {

	assert.Equal(t, "foo%20bar", getPrefixFromPath("/foo%20bar/glum"))
	assert.Equal(t, "", getPrefixFromPath("ooops/foo%20bar/glum"))

}

func TestAPI(t *testing.T) {

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

	closed := make(chan struct{})
	var wg sync.WaitGroup

	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	secret := "testsecret"

	audience := "http://[::]:" + strconv.Itoa(port)
	cs := ttlcode.NewDefaultCodeStore()
	target := "wss://relay.example.io"

	wg.Add(1)

	go API(closed, &wg, port, audience, secret, target, cs)

	time.Sleep(100 * time.Millisecond)

	client := &http.Client{}

	// Start tests
	req, err := http.NewRequest("POST", audience+"/session/123", nil)
	assert.NoError(t, err)
	resp, err := client.Do(req)
	assert.NoError(t, err)
	body, _ := ioutil.ReadAll(resp.Body)
	bodyStr := string([]byte(body))
	assert.Equal(t, `{"code":401,"message":"unauthenticated for invalid credentials"}`, bodyStr)

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

	req, err = http.NewRequest("POST", audience+"/session/123", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)

	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, _ = ioutil.ReadAll(resp.Body)

	var p operations.SessionOKBody
	err = json.Unmarshal(body, &p)
	assert.NoError(t, err)

	expected := "wss://relay.example.io/session/123?code="

	assert.Equal(t, expected, p.URI[0:len(expected)])

	// End tests
	close(closed)
	wg.Wait()

}
