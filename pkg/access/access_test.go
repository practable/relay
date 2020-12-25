package access

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
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
	"github.com/timdrysdale/relay/pkg/ttlcode"
)

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

	go API(closed, &wg, port, audience, secret, target, cs, Options{})

	time.Sleep(100 * time.Millisecond)

	client := &http.Client{}

	// Start tests
	req, err := http.NewRequest("POST", audience+"/session/123", nil)

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
	claims.ConnectionType = "session"

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign and get the complete encoded token as a string using the secret
	bearer, err := token.SignedString([]byte(secret))
	assert.NoError(t, err)

	req, err = http.NewRequest("POST", audience+"/session/123", nil)
	req.Header.Add("Authorization", bearer)
	t.Log(bearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, _ = ioutil.ReadAll(resp.Body)
	bodyStr = string([]byte(body))
	t.Log(bodyStr)

	// End tests
	close(closed)
	wg.Wait()

}
