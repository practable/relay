package shellhost

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/timdrysdale/relay/pkg/permission"
	"github.com/timdrysdale/relay/pkg/reconws"
	"github.com/timdrysdale/relay/pkg/shellaccess"
	"github.com/timdrysdale/relay/pkg/shellbar"
	"github.com/timdrysdale/relay/pkg/shellrelay"
	"github.com/timdrysdale/relay/pkg/tcpconnect"
)

func init() {

	debug := true
	if debug {
		log.SetReportCaller(true)
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true, DisableColors: true})
		defer log.SetOutput(os.Stdout)

	} else {
		var ignore bytes.Buffer
		logignore := bufio.NewWriter(&ignore)
		log.SetOutput(logignore)
	}
}

func TestShellhost(t *testing.T) {

	// Setup logging

	timeout := 100 * time.Millisecond

	// setup shellrelay on local (free) port
	closed := make(chan struct{})
	var wg sync.WaitGroup

	ports, err := freeport.GetFreePorts(2)
	assert.NoError(t, err)

	relayPort := ports[0]
	accessPort := ports[1]

	shellaccessURI := "http://[::]:" + strconv.Itoa(accessPort)
	shellrelayURI := "ws://127.0.0.1:" + strconv.Itoa(relayPort)

	log.Debug(fmt.Sprintf("shellaccessURI:%s\n", shellaccessURI))
	log.Debug(fmt.Sprintf("shellrelayURI:%s\n", shellrelayURI))

	secret := "testsecret"

	wg.Add(1)

	go shellrelay.Relay(closed, &wg, accessPort, relayPort, shellaccessURI, secret, shellrelayURI, shellaccess.Options{})

	// setup mock sshd
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	shellport, err := freeport.GetFreePort()
	if err != nil {
		t.Fatal(err)
	}

	sshduri := ":" + strconv.Itoa(shellport)

	echo := tcpconnect.New()
	go echo.Echo(ctx, sshduri)

	time.Sleep(time.Second)
	time.Sleep(timeout)
	time.Sleep(timeout)
	time.Sleep(timeout)
	time.Sleep(timeout)

	// setup shellhost
	ct := "shell"
	session := "11014d77-e36e-40b7-9864-5a9239d1a071"
	scopes := []string{"host"} //host, client scopes are known only to access

	begin := time.Now().Unix() - 1 //ensure it's in the past
	end := begin + 180
	claims := permission.NewToken(shellaccessURI, ct, session, scopes, begin, begin, end)
	hostToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	hostBearer, err := hostToken.SignedString([]byte(secret))
	assert.NoError(t, err)

	go Shellhost(ctx, "localhost"+sshduri, shellaccessURI+"/shell/"+session, hostBearer)

	time.Sleep(time.Second)
	time.Sleep(timeout)
	time.Sleep(timeout)
	time.Sleep(timeout)
	time.Sleep(timeout)

	// ============================= START  TESTS ======================================

	// *** TestConnectToLocalShell ***

	scopes = []string{"client"} //host, client scopes are known only to access
	claims = permission.NewToken(shellaccessURI, "shell", session, scopes, begin, begin, end)
	clientToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	clientBearer, err := clientToken.SignedString([]byte(secret))

	clientURI := shellaccessURI + "/shell/" + session

	var c0 *reconws.ReconWs

	c0 = reconws.New()
	go c0.ReconnectAuth(ctx, clientURI, clientBearer)

	c1 := reconws.New()
	go c1.ReconnectAuth(ctx, clientURI, clientBearer)

	// Send messages, get echos...
	time.Sleep(3 * time.Second) //give shellhost a chance to make new connections

	time.Sleep(timeout)
	time.Sleep(timeout)
	time.Sleep(timeout)
	time.Sleep(timeout)
	time.Sleep(timeout)
	time.Sleep(timeout)

	data0 := []byte("ping")

	select {
	case <-time.After(timeout):
		t.Fatal("timeout")
	case c0.Out <- reconws.WsMessage{Data: data0, Type: websocket.BinaryMessage}:

	}

	time.Sleep(timeout)
	time.Sleep(timeout)
	time.Sleep(timeout)

	select {
	case <-time.After(timeout):
		t.Fatal("timeout")
	case msg, ok := <-c0.In:
		assert.True(t, ok)
		assert.Equal(t, data0, msg.Data)
		t.Log("TestConnectToLocalShell...PASS")
	}

	select {
	case <-time.After(timeout):
	case <-c1.In:
		t.Fatal("unexpected")
	}

	data1 := []byte("foo")
	select {
	case <-time.After(timeout):
		t.Fatal("timeout")
	case c1.Out <- reconws.WsMessage{Data: data1, Type: websocket.BinaryMessage}:
	}

	select {
	case <-time.After(timeout):
		t.Fatal("timeout")
	case msg, ok := <-c1.In:
		assert.True(t, ok)
		assert.Equal(t, data1, msg.Data)
	}

	select {
	case <-time.After(timeout):
	case <-c0.In:
		t.Fatal("unexpected")
	}

	// while connected, get stats
	scopes = []string{"stats"}
	claims = permission.NewToken(shellaccessURI, "shell", "stats", scopes, begin, begin, end)
	statsToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	statsBearer, err := statsToken.SignedString([]byte(secret))

	stats := reconws.New()
	go stats.ReconnectAuth(ctx, shellaccessURI+"/shell/stats", statsBearer)

	cmd, err := json.Marshal(shellbar.StatsCommand{Command: "update"})

	assert.NoError(t, err)

	stats.Out <- reconws.WsMessage{Data: cmd, Type: websocket.TextMessage}

	select {
	case msg := <-stats.In:

		var reports []*shellbar.ClientReport

		err := json.Unmarshal(msg.Data, &reports)

		assert.NoError(t, err)

		agents := make(map[string]int)

		for _, report := range reports {
			count, ok := agents[report.Topic]
			if !ok {
				agents[report.Topic] = 1
				continue
			}

			agents[report.Topic] = count + 1
		}

		sessionCount := 0
		for topic, count := range agents {
			log.Debug(topic)
			if strings.HasPrefix(topic, session+"/") {
				sessionCount = sessionCount + count
			}
		}
		expectedCount := 4
		assert.Equal(t, expectedCount, sessionCount)

		//TODO we can't know this because salted, so search for partial match to session
		if sessionCount == expectedCount {
			t.Log("TestGetStats...PASS")
		} else {
			pretty, err := json.MarshalIndent(reports, "", "\t")
			assert.NoError(t, err)
			t.Log(string(pretty))
			t.Fatalf("TestGetStats...FAIL (wrong agent count)")
		}

	case <-time.After(timeout):
		t.Fatalf("TestGetStats...FAIL (timeout)")
	}

	// ================================== Teardown  ===============================================
	cancel()
	time.Sleep(timeout)
	close(closed)
	wg.Wait()

}
