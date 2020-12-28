package shellbar

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/phayes/freeport"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/timdrysdale/reconws"
	"github.com/timdrysdale/relay/pkg/permission"
	"github.com/timdrysdale/relay/pkg/ttlcode"
)

// NOTE don't use reconws.Reconnect for production clients anymore;
// it does NOT understand the use of auth codes
// use Dial instead.

func MakeTestToken(audience, connectionType, topic string, scopes []string, lifetime int64) permission.Token {
	begin := time.Now().Unix() - 1 //ensure it's in the past
	end := begin + lifetime
	return permission.NewToken(audience, connectionType, topic, scopes, begin, begin, end)
}

func TestShellbar(t *testing.T) {

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

	// Setup shellbar

	http.DefaultServeMux = new(http.ServeMux)

	// setup shellbar on local (free) port
	closed := make(chan struct{})
	var wg sync.WaitGroup

	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	audience := "ws://127.0.0.1:" + strconv.Itoa(port)
	secret := "somesecret"
	cs := ttlcode.NewDefaultCodeStore()
	config := Config{
		Listen:    port,
		Audience:  audience,
		CodeStore: cs,
		Secret:    secret,
	}

	wg.Add(1)
	go Shellbar(config, closed, &wg)
	// safety margin to get shellbar running
	time.Sleep(time.Second)

	var timeout = 100 * time.Millisecond

	// Start tests

	// *** TestConnectUniquely ***

	ctx, cancel := context.WithCancel(context.Background())

	// construct host token & connect
	ct := "shell"
	session := "20fd9a71-2248-4f60-89e3-5d5bb2e78e09"
	scopes := []string{"read", "write"} //host, client scopes are known only to access

	tokenHost := MakeTestToken(audience, ct, session, scopes, 5)
	codeHost := cs.SubmitToken(tokenHost)

	h := reconws.New()
	go h.Dial(ctx, audience+"/"+ct+"/"+session+"?code="+codeHost)

	// construct client token & connect
	connectionID := "607109e7-3841-4d0f-ba6a-91e8b817f3f5"
	clientTopic := session + "/" + connectionID
	topicSalt := "42f3b247-2632-40b3-8aeb-ce2218cbf26d"
	topicInHub := clientTopic + topicSalt
	tokenClient := MakeTestToken(audience, ct, clientTopic, scopes, 5)
	permission.SetTopicSalt(&tokenClient, topicSalt)
	permission.SetAlertHost(&tokenClient, true)

	codeClient0 := cs.SubmitToken(tokenClient)
	c0 := reconws.New()
	client0UniqueURI := audience + "/" + ct + "/" + clientTopic

	ctx0, cancel0 := context.WithCancel(context.Background())
	go c0.Dial(ctx0, client0UniqueURI+"?code="+codeClient0)

	var ca ConnectionAction

	var c0UUID string
	select {

	case <-time.After(time.Second):
		t.Fatal("TestHostAdminGetsConnectAction...FAIL\n")

	case msg, ok := <-h.In:

		assert.True(t, ok)

		err = json.Unmarshal(msg.Data, &ca)
		assert.NoError(t, err)
		assert.Equal(t, "connect", ca.Action)

		base := strings.Split(ca.URI, "?")[0]
		c0UUID = ca.UUID
		assert.Equal(t, client0UniqueURI, base)
		if client0UniqueURI == base {
			t.Logf("TestHostAdminGetsConnectAction...PASS\n")
		} else {
			t.Fatal("TestHostAdminGetsConnectAction...FAIL\n")
		}
	}

	// Host now dials the unqiue connection
	h1 := reconws.New()
	go h1.Dial(ctx, ca.URI)

	time.Sleep(timeout)

	data := []byte("ping")

	h1.Out <- reconws.WsMessage{Data: data, Type: websocket.TextMessage}

	select {
	case msg := <-c0.In:
		assert.Equal(t, data, msg.Data)
		if reflect.DeepEqual(data, msg.Data) {
			t.Logf("TestHostConnectsToUniqueSession...PASS\n")
		} else {
			t.Fatal("TestHostConnectsToUniqueSession...FAIL")
		}
	case <-time.After(timeout):
		t.Fatal("TestHostConnectsToUniqueSession...FAIL")
	}

	data = []byte("pong")

	c0.Out <- reconws.WsMessage{Data: data, Type: websocket.TextMessage}
	select {
	case msg := <-h1.In:
		assert.Equal(t, data, msg.Data)
		if reflect.DeepEqual(data, msg.Data) {
			t.Logf("TestHostReceivesDataFromUniqueSession...PASS\n")
		} else {
			t.Fatal("TestHostReceivesDataFromUniqueSession...FAIL (wrong message)")
		}
	case <-time.After(timeout):
		t.Fatal("TestHostReceivesDataFromUniqueSession...FAIL")
	}

	// while connected, get stats
	scopes = []string{"read", "write"}
	statsToken := MakeTestToken(audience, ct, "stats", scopes, 5)
	statsCode := cs.SubmitToken(statsToken)
	stats := reconws.New()
	go stats.Dial(ctx, audience+"/"+ct+"/stats?code="+statsCode)

	cmd, err := json.Marshal(StatsCommand{Command: "update"})

	assert.NoError(t, err)

	stats.Out <- reconws.WsMessage{Data: cmd, Type: websocket.TextMessage}

	select {
	case msg := <-stats.In:

		t.Log("TestGetStats...PROVISIONAL-PASS")

		var reports []*ClientReport

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

		if agents[topicInHub] == 2 {
			t.Log("TestGetStats...PASS")
		} else {
			t.Fatalf("TestGetStats...FAIL")
			pretty, err := json.MarshalIndent(reports, "", "\t")
			assert.NoError(t, err)
			fmt.Println(string(pretty))
		}

	case <-time.After(timeout):
		t.Fatalf("TestGetStats...FAIL")
	}

	time.Sleep(timeout)

	cancel0()

	time.Sleep(timeout)
	time.Sleep(timeout)
	time.Sleep(timeout)
	time.Sleep(timeout)
	select {
	case <-time.After(time.Second):
		t.Fatal("No disconnect message")
	case msg, ok := <-h.In:
		assert.True(t, ok)

		err = json.Unmarshal(msg.Data, &ca)
		assert.NoError(t, err)
		assert.Equal(t, "disconnect", ca.Action)
		assert.Equal(t, c0UUID, ca.UUID)

		if c0UUID == ca.UUID {
			t.Logf("TestHostAdminGetsDisconnectAction...PASS\n")
		} else {
			t.Fatal("TestHostAdminGetsDisconnectAction...FAIL\n")
		}

	}

	time.Sleep(timeout)
	cancel()
	// Teardown crossbar
	time.Sleep(timeout)
	close(closed)
	wg.Wait()

}
