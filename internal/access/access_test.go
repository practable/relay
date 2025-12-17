package access

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/phayes/freeport"
	"github.com/practable/relay/internal/access/models"
	"github.com/practable/relay/internal/access/restapi/operations"
	"github.com/practable/relay/internal/crossbar"
	"github.com/practable/relay/internal/deny"
	"github.com/practable/relay/internal/permission"
	"github.com/practable/relay/internal/ttlcode"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestGetPrefixFromPath(t *testing.T) {

	assert.Equal(t, "foo%20bar", getPrefixFromPath("/foo%20bar/glum"))
	assert.Equal(t, "", getPrefixFromPath("ooops/foo%20bar/glum"))

}

func TestAPI(t *testing.T) {

	debug := false

	if debug {
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true, DisableColors: true})
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

	ds := deny.New()

	config := Config{
		AllowNoBookingID: true, //backwards compatible test
		CodeStore:        cs,
		DenyStore:        ds,
		Host:             audience,
		Port:             port,
		Secret:           secret,
		Target:           target,
	}

	go API(closed, &wg, config) //port, audience, secret, target, cs, ds, allowNoBookingID)

	time.Sleep(100 * time.Millisecond)

	client := &http.Client{}

	// Start tests
	req, err := http.NewRequest("POST", audience+"/session/123", nil)
	assert.NoError(t, err)
	resp, err := client.Do(req)
	assert.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
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
	body, _ = io.ReadAll(resp.Body)

	var p operations.SessionOKBody
	err = json.Unmarshal(body, &p)
	assert.NoError(t, err)

	expected := "wss://relay.example.io/session/123?code="

	assert.Equal(t, expected, p.URI[0:len(expected)])

	// End tests
	close(closed)
	wg.Wait()

}

func TestGetStats(t *testing.T) {

	debug := false

	if debug {
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true, DisableColors: true})
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
	hub := crossbar.New()

	wg.Add(1)

	ds := deny.New()

	config := Config{
		AllowNoBookingID: true, //backwards compatible test
		CodeStore:        cs,
		DenyStore:        ds,
		Host:             audience,
		Hub:              hub,
		Port:             port,
		Secret:           secret,
		Target:           target,
	}

	go API(closed, &wg, config) //port, audience, secret, target, cs, ds, allowNoBookingID)

	time.Sleep(100 * time.Millisecond)

	client := &http.Client{}

	// Start tests
	req, err := http.NewRequest("GET", audience+"/status", nil)
	assert.NoError(t, err)
	resp, err := client.Do(req)
	assert.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string([]byte(body))
	assert.Equal(t, `{"code":401,"message":"unauthenticated for invalid credentials"}`, bodyStr)

	var claims permission.Token

	start := jwt.NewNumericDate(time.Now().Add(-time.Second))
	after5 := jwt.NewNumericDate(time.Now().Add(5 * time.Second))

	claims.IssuedAt = start
	claims.NotBefore = start
	claims.ExpiresAt = after5
	claims.Audience = jwt.ClaimStrings{audience}
	claims.Scopes = []string{"relay:stats"}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign and get the complete encoded token as a string using the secret
	bearer, err := token.SignedString([]byte(secret))
	assert.NoError(t, err)

	req, err = http.NewRequest("GET", audience+"/status", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)

	_, err = client.Do(req)
	assert.NoError(t, err)

	// this test does not check for a meangingful stats response,
	// merely that the method can be called

	// End tests
	close(closed)
	wg.Wait()

}

func TestDeny(t *testing.T) {

	debug := false

	if debug {
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true, DisableColors: true})
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

	ds := deny.New()

	dc := make(chan string, 2) //probably larger buffer needed in production

	denied := []string{}

	go func() { //collect any denials sent
		for {
			select {
			case d := <-dc:
				denied = append(denied, d)
			case <-closed:
				return
			}
		}
	}()

	config := Config{
		AllowNoBookingID: false, //reject tokens without bookingID
		CodeStore:        cs,
		DenyChannel:      dc,
		DenyStore:        ds,
		Host:             audience,
		Port:             port,
		Secret:           secret,
		Target:           target,
	}

	go API(closed, &wg, config) //port, audience, secret, target, cs, ds, allowNoBookingID)

	time.Sleep(100 * time.Millisecond)

	client := &http.Client{}

	// Start tests
	req, err := http.NewRequest("POST", audience+"/session/123", nil)
	assert.NoError(t, err)
	resp, err := client.Do(req)
	assert.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
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

	// make a request, which will fail, because we have no bookingID!
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, _ = io.ReadAll(resp.Body)

	bodyStr = string([]byte(body))
	assert.Equal(t, `{"code":"400","message":"empty bookingID field is not permitted"}`+"\n", bodyStr) //not sure why this one needs a return at the end of the line when the other doesn't ... but it's what works.

	// Try again with a bookingID

	claims.BookingID = "some-bid"
	token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	bearer, err = token.SignedString([]byte(secret))
	assert.NoError(t, err)

	req, err = http.NewRequest("POST", audience+"/session/123", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)

	// make a request, which will succedd now we have a bookingID!
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, _ = io.ReadAll(resp.Body)

	var p operations.SessionOKBody
	err = json.Unmarshal(body, &p)
	assert.NoError(t, err)

	expected := "wss://relay.example.io/session/123?code="

	assert.Equal(t, expected, p.URI[0:len(expected)])

	// Deny a booking ID

	// create an admin token
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
	q.Add("bid", "another-bid")
	q.Add("exp", strconv.Itoa(int(time.Now().Unix()+5)))
	req.URL.RawQuery = q.Encode()
	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)

	// Try to get a connection for the denied bookingID
	claims.BookingID = "another-bid"
	token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	bearer, err = token.SignedString([]byte(secret))
	assert.NoError(t, err)

	req, err = http.NewRequest("POST", audience+"/session/123", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)

	// make a request, which will fail, because our bookingID has been denied
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, _ = io.ReadAll(resp.Body)

	bodyStr = string([]byte(body))
	assert.Equal(t, `{"code":"400","message":"bookingID has been deny-listed, probably because the session was cancelled"}`+"\n", bodyStr)

	// Get the DeniedList
	req, err = http.NewRequest("GET", audience+"/bids/deny", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, _ = io.ReadAll(resp.Body)
	var db models.BookingIDs

	err = json.Unmarshal(body, &db)
	assert.NoError(t, err)

	de := []string{"another-bid"}
	assert.Equal(t, de, db.BookingIds)

	// Get the AllowedList
	req, err = http.NewRequest("GET", audience+"/bids/allow", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, _ = io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &db)
	assert.NoError(t, err)

	ae := []string{"some-bid"}
	assert.Equal(t, ae, db.BookingIds)

	// check the deny channel got the right messages
	assert.Equal(t, de, denied)

	// Undo the earlier denial
	req, err = http.NewRequest("POST", audience+"/bids/allow", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	q = req.URL.Query()
	q.Add("bid", "another-bid")
	q.Add("exp", strconv.Itoa(int(time.Now().Unix()+5)))
	req.URL.RawQuery = q.Encode()
	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)

	// Try getting a connection now we're allowed again - should succeed
	claims.BookingID = "another-bid"
	token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	bearer, err = token.SignedString([]byte(secret))
	assert.NoError(t, err)

	req, err = http.NewRequest("POST", audience+"/session/123", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)

	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, _ = io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &p)
	assert.NoError(t, err)
	expected = "wss://relay.example.io/session/123?code="
	assert.Equal(t, expected, p.URI[0:len(expected)])

	// Get the AllowedList
	req, err = http.NewRequest("GET", audience+"/bids/allow", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, _ = io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &db)
	assert.NoError(t, err)

	ae = []string{"some-bid", "another-bid"}

	// results could come in any order so make a map
	aem := make(map[string]bool)
	aam := make(map[string]bool)

	for _, v := range ae {
		aem[v] = true
	}
	for _, v := range db.BookingIds {
		aam[v] = true
	}

	assert.Equal(t, aem, aam)

	// Get the DeniedList
	req, err = http.NewRequest("GET", audience+"/bids/deny", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, _ = io.ReadAll(resp.Body)

	err = json.Unmarshal(body, &db)
	assert.NoError(t, err)

	de = []string{}
	assert.Equal(t, de, db.BookingIds)

	// End tests
	close(closed)
	wg.Wait()

}

func TestBadBearerHandledOK(t *testing.T) {

	debug := false

	if debug {
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true, DisableColors: true})
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

	ds := deny.New()

	dc := make(chan string, 2) //probably larger buffer needed in production

	go func() { //collect any denials sent
		for {
			select {
			case <-dc:
				continue // just drain the channel, not testing deny channel stuff here
			case <-closed:
				return
			}
		}
	}()

	config := Config{
		AllowNoBookingID: true, //reject tokens without bookingID
		CodeStore:        cs,
		DenyChannel:      dc,
		DenyStore:        ds,
		Host:             audience,
		Port:             port,
		Secret:           secret,
		Target:           target,
	}

	go API(closed, &wg, config) //port, audience, secret, target, cs, ds, allowNoBookingID)

	time.Sleep(100 * time.Millisecond)

	client := &http.Client{}

	// Start tests
	req, err := http.NewRequest("POST", audience+"/session/123", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", "eyasdfasd.asdfasdf.asdf4325") //nonsense bearer token

	// make a request, which will fail, because the bearer token is bogus
	resp, err := client.Do(req)
	assert.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	bodyStr := string([]byte(body))
	assert.Equal(t, `{"code":500,"message":"token invalid"}`, bodyStr)

	// make a request, which will fail, because the bearer token is empty
	req, err = http.NewRequest("POST", audience+"/session/123", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", "") //empty bearer token

	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, _ = io.ReadAll(resp.Body)

	bodyStr = string([]byte(body))
	assert.Equal(t, `{"code":401,"message":"unauthenticated for invalid credentials"}`, bodyStr)
	_ = resp.Body.Close()

	// End tests
	close(closed)
	wg.Wait()

}
