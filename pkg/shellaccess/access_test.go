package shellaccess

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/phayes/freeport"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/timdrysdale/relay/pkg/permission"
	"github.com/timdrysdale/relay/pkg/shellaccess/restapi/operations"
	"github.com/timdrysdale/relay/pkg/ttlcode"
)

func TestGetPrefixFromPath(t *testing.T) {

	assert.Equal(t, "foo%20bar", getPrefixFromPath("/foo%20bar/glum"))
	assert.Equal(t, "", getPrefixFromPath("ooops/foo%20bar/glum"))

}

func TestTokenGeneration(t *testing.T) {

	iat := int64(1609329233)
	nbf := int64(1609329233)
	exp := int64(1609330233)
	audience := "https://relay-access.example.io"
	ct := "shell"
	topic := "f7558de0-cb0d-4cb5-9518-ac71d044800b"
	scopes := []string{"host"}
	secret := "somesecret"

	expected := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0b3BpYyI6ImY3NTU4ZGUwLWNiMGQtNGNiNS05NTE4LWFjNzFkMDQ0ODAwYiIsInByZWZpeCI6InNoZWxsIiwic2NvcGVzIjpbImhvc3QiXSwiYXVkIjoiaHR0cHM6Ly9yZWxheS1hY2Nlc3MuZXhhbXBsZS5pbyIsImV4cCI6MTYwOTMzMDIzMywiaWF0IjoxNjA5MzI5MjMzLCJuYmYiOjE2MDkzMjkyMzN9.fAuyD3buu1XWV8qLojEDTImZf_IrJXEu5g3oziKbZbY"

	bearer, err := Token(audience, ct, topic, secret, scopes, iat, nbf, exp)

	assert.NoError(t, err)
	assert.Equal(t, expected, bearer)

}

func TestAPI(t *testing.T) {

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
	req, err := http.NewRequest("POST", audience+"/shell/123", nil)
	assert.NoError(t, err)
	resp, err := client.Do(req)
	assert.NoError(t, err)
	body, _ := ioutil.ReadAll(resp.Body)
	bodyStr := string([]byte(body))
	assert.Equal(t, `{"code":401,"message":"unauthenticated for invalid credentials"}`, bodyStr)

	var claims permission.Token

	claims.IssuedAt = time.Now().Unix() - 1
	claims.NotBefore = time.Now().Unix() - 1
	claims.ExpiresAt = time.Now().Unix() + 5
	claims.Audience = audience
	claims.Topic = "123"
	claims.ConnectionType = "shell"
	claims.Scopes = []string{"read", "write"} //Wrong scopes for shell, deliberately....

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign and get the complete encoded token as a string using the secret
	bearer, err := token.SignedString([]byte(secret))
	assert.NoError(t, err)

	req, err = http.NewRequest("POST", audience+"/shell/123", nil)
	req.Header.Add("Authorization", bearer)

	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, _ = ioutil.ReadAll(resp.Body)

	assert.Equal(t, "\"Missing client, host or stats Scope\"\n", string(body))

	// now try with correct scopes :-)
	claims.Scopes = []string{"host"}
	token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	bearer, err = token.SignedString([]byte(secret))
	assert.NoError(t, err)

	req, err = http.NewRequest("POST", audience+"/shell/123", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)

	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, _ = ioutil.ReadAll(resp.Body)

	var p operations.ShellOKBody

	err = json.Unmarshal(body, &p)
	assert.NoError(t, err)
	if err != nil {
		t.Log(string(body))
	}

	expected := "wss://relay.example.io/shell/123?code="

	if len(p.URI) < len(expected) {
		t.Fatal("URI too short")
	} else {
		assert.Equal(t, expected, p.URI[0:len(expected)])
	}

	// Now repeat with client, expecting to get a connectionID added to the uri ...

	claims.Scopes = []string{"client"}
	token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	bearer, err = token.SignedString([]byte(secret))
	assert.NoError(t, err)

	req, err = http.NewRequest("POST", audience+"/shell/123", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)

	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, _ = ioutil.ReadAll(resp.Body)

	err = json.Unmarshal(body, &p)
	assert.NoError(t, err)
	if err != nil {
		t.Log(string(body))
	}

	expected = "wss://relay.example.io/shell/123/"

	if len(p.URI) < len(expected) {
		t.Fatal("URI too short")
	} else {
		assert.Equal(t, expected, p.URI[0:len(expected)])
	}

	re := regexp.MustCompile("wss:\\/\\/relay\\.example\\.io\\/shell\\/123\\/([\\w-\\%]*)\\?code=.*")
	matches := re.FindStringSubmatch(p.URI)
	assert.Equal(t, 36, len(matches[1])) //length of a UUID

	uniqueConnection0 := matches[1]

	// repeat the request, and check we get a different connection id

	req, err = http.NewRequest("POST", audience+"/shell/123", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)

	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, _ = ioutil.ReadAll(resp.Body)

	err = json.Unmarshal(body, &p)
	assert.NoError(t, err)
	if err != nil {
		t.Log(string(body))
	}

	expected = "wss://relay.example.io/shell/123/"

	if len(p.URI) < len(expected) {
		t.Fatal("URI too short")
	} else {
		assert.Equal(t, expected, p.URI[0:len(expected)])
	}

	matches = re.FindStringSubmatch(p.URI)
	assert.Equal(t, 36, len(matches[1])) //length of a UUID

	assert.NotEqual(t, uniqueConnection0, matches[1])

	// End tests
	close(closed)
	wg.Wait()
}
