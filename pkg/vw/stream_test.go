package vw

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/phayes/freeport"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/timdrysdale/relay/pkg/agg"
	"github.com/timdrysdale/relay/pkg/permission"
	"github.com/timdrysdale/relay/pkg/reconws"
	"github.com/timdrysdale/relay/pkg/relay"
	"github.com/timdrysdale/relay/pkg/rwc"
)

func TestStreamUsingInternals(t *testing.T) {
	//
	//  This integration test is intended to show a file streaming
	//  to a websocket server, using elements of existing tests
	//  but joined together this time ...
	//
	//  +------+   +------+   +------+    +------+    +------+
	//  |      |   |      |   |      |    |      |    |      |
	//  |ffmpeg+--->handle+--->Agg   +---->rwc   +--->+ wss  |
	//  |      |   |Ts    |   |      |    |      |    |      |
	//  +-^----+   +------+   +-^----+    +-^----+    +-----++
	//    |                     |           |               |
	//    |                     |           |               |
	//    +                     +           +               v
	//  sample.ts             stream      destination    check
	//                        rule        rule           frame
	//                                                   sizes
	//

	// start up our streaming programme
	//go streamCmd.Run(streamCmd, nil) //streamCmd will populate the global app
	app := testApp(true)

	time.Sleep(2 * time.Millisecond)

	// server to action the handler under test
	r := mux.NewRouter()
	r.HandleFunc("/ts/{feed}", http.HandlerFunc(app.handleTs))

	s := httptest.NewServer(r)
	defer s.Close()

	time.Sleep(100 * time.Millisecond)

	// Set up our destination wss server and frame size check

	msgSize := make(chan int)

	serverExternal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reportSize(w, r, msgSize) }))
	defer serverExternal.Close()

	go func() {
		// did frame sizes come through correctly?
		frameSizes := []int{15980,
			20116,
			17296,
			16544,
			18988,
		}

		time.Sleep(100 * time.Millisecond) //give ffmpeg time to start before looking for frames

		for i := 0; i < len(frameSizes); i++ {
			select {
			case <-time.After(100 * time.Millisecond):
				t.Errorf("timed out on frame  %d", i)
			case frameSize, ok := <-msgSize:
				if ok {
					if frameSize != frameSizes[i] {
						t.Errorf("Frame size %d  wrong; got/wanted %v/%v\n", i, frameSize, frameSizes[i])
					}
				} else {
					t.Error("channel not ok")
				}
			}
		}
	}()

	time.Sleep(1 * time.Millisecond)

	// set up our rules (we've not got audio, but use stream for more thorough test
	streamRule := agg.Rule{Stream: "stream/large", Feeds: []string{"video0", "audio"}}
	app.Hub.Add <- streamRule

	ue, _ := url.Parse(serverExternal.URL)
	wssURL := fmt.Sprintf("ws://localhost:%s", ue.Port())
	destinationRule := rwc.Rule{Stream: "stream/large", Destination: wssURL, ID: "00"}
	app.Websocket.Add <- destinationRule

	time.Sleep(1 * time.Millisecond)

	uv, _ := url.Parse(s.URL)
	dest := fmt.Sprintf("http://localhost:%s/ts/video0", uv.Port())
	//dest := "http://localhost:8888/ts/video"
	args := fmt.Sprintf("-re -i sample.ts -f mpegts -codec:v mpeg1video -s 640x480 -b:v 1000k -r 24 -bf 0 %s", dest)
	argSlice := strings.Split(args, " ")
	cmd := exec.Command("ffmpeg", argSlice...)
	fp, err := filepath.Abs("./test")
	assert.NoError(t, err)
	cmd.Dir = fp
	err = cmd.Run()
	if err != nil {
		t.Error("ffmpeg", err)
	}

	// hang on long enough for timeouts in the anonymous goroutine to trigger
	time.Sleep(300 * time.Millisecond)

	close(app.Closed)

	time.Sleep(time.Millisecond) //allow time for goroutines to end before starting a new http server

}

func TestStreamUsingStreamCmd(t *testing.T) {
	//
	//  This integration test is intended to show a file streaming
	//  to a websocket server, using elements of existing tests
	//  but joined together this time ...
	//
	//  +------+   +------+   +------+    +------+    +------+
	//  |      |   |      |   |      |    |      |    |      |
	//  |ffmpeg+--->handle+--->Agg   +---->rwc   +--->+ wss  |
	//  |      |   |Ts    |   |      |    |      |    |      |
	//  +-^----+   +------+   +-^----+    +-^----+    +-----++
	//    |                     |           |               |
	//    |                     |           |               |
	//    +                     +           +               v
	//  sample.ts             stream      destination    check
	//                        rule        rule           frame
	//                                                   sizes
	//
	// start up our streaming programme
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
	//go streamCmd.Run(streamCmd, nil) //streamCmd will populate the global app
	go Stream()

	// destination websocket reporting channels
	msgSize0 := make(chan int)
	msgSize1 := make(chan int)

	//destination websocket servers
	serverExternal0 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reportSize(w, r, msgSize0) }))
	defer serverExternal0.Close()

	serverExternal1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reportSize(w, r, msgSize1) }))
	defer serverExternal1.Close()

	// destination changeover signalling (to save guessing ffmpeg startup time)
	ffmpegRunning := make(chan struct{})

	// stream rule (we've not got audio, but use stream for more thorough test)
	streamRule := agg.Rule{Stream: "stream/large", Feeds: []string{"video0", "audio"}}

	// destination rules
	ID := "00" //same for both as changing, not duplicating, destination

	url0, _ := url.Parse(serverExternal0.URL)
	wss0 := fmt.Sprintf("ws://localhost:%s", url0.Port())
	destinationRule0 := rwc.Rule{Stream: "stream/large", Destination: wss0, ID: ID}

	url1, _ := url.Parse(serverExternal1.URL)
	wss1 := fmt.Sprintf("ws://localhost:%s", url1.Port())
	destinationRule1 := rwc.Rule{Stream: "stream/large", Destination: wss1, ID: ID}

	// receivers
	rxCount0 := 0
	rxCount1 := 0
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		select {
		case <-msgSize0:
			close(ffmpegRunning) //signal to main body of test
			rxCount0++
		case <-time.After(time.Second):
			t.Error("Timeout on destination websocket server 0")
		}
		for msgSize := range msgSize0 {
			if msgSize > 0 {
				rxCount0++
			}
		}
	}()
	go func() {
		defer wg.Done()
		select {
		case <-msgSize1:
			rxCount1++
		case <-time.After(time.Second):
			t.Error("Timeout on destination websocket server 1")
		}
		for msgSize := range msgSize1 {
			if msgSize > 0 {
				rxCount1++
			}
		}
	}()

	app.Hub.Add <- streamRule
	app.Websocket.Add <- destinationRule0

	time.Sleep(1 * time.Millisecond)

	// start ffmpeg; use default vw port
	dest := "http://localhost:8888/ts/video0"
	args := fmt.Sprintf("-re -f concat -i list.txt -f mpegts -codec:v mpeg1video -s 640x480 -b:v 1000k -r 24 -bf 0 %s", dest)
	argSlice := strings.Split(args, " ")
	cmd := exec.Command("ffmpeg", argSlice...)
	fp, err := filepath.Abs("./test")
	assert.NoError(t, err)
	cmd.Dir = fp
	go func() {
		err := cmd.Run()
		if err != nil {
			t.Error("ffmpeg", err)
		}
	}()

	select {
	case <-ffmpegRunning:
	case <-time.After(time.Second): //avoid hanging if test failed
		t.Error("Timeout: ffmpeg too slow to send first frame / ffmpeg frames not received")
	}

	// wait five frames
	time.Sleep(5 * 42 * time.Millisecond)

	app.Websocket.Add <- destinationRule1

	// wait another 40 frames
	time.Sleep(40 * 42 * time.Millisecond)

	if rxCount0 < 5 {
		t.Errorf("Insufficient frames received by server 0: %d", rxCount0)
	}

	if rxCount1 < 5 {
		t.Errorf("Insufficient frames received by server 1: %d", rxCount1)
	}

	close(app.Closed)

	close(msgSize0)
	close(msgSize1)

	time.Sleep(1 * time.Second)
}

func TestStreamUsingStreamCmdAuth(t *testing.T) {
	//
	//  This integration test is intended to show a file streaming
	//  to a websocket server, using elements of existing tests
	//  but joined together this time ... with JWT Auth this time
	//
	//  +------+   +------+   +------+    +------+    +------+
	//  |      |   |      |   |      |    |      |    |      |
	//  |ffmpeg+--->handle+--->Agg   +---->rwc   +--->+ wss  |
	//  |      |   |Ts    |   |      |    |      |    |      |
	//  +-^----+   +------+   +-^----+    +-^----+    +-----++
	//    |                     |           |               |
	//    |                     |           |               |
	//    +                     +           +               v
	//  sample.ts             stream      destination    check
	//                        rule        rule           frame
	//                                                   sizes
	//
	// start up our streaming programme

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

	// Setup relay on local (free) ports to receive wss streams
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

	go relay.Relay(closed, &wg, accessPort, relayPort, audience, secret, target)

	//go streamCmd.Run(streamCmd, nil) //streamCmd will populate the global app
	go Stream()

	time.Sleep(time.Second) // big safety margin to get crossbar running

	// destination changeover signalling (to save guessing ffmpeg startup time)
	ffmpegRunning := make(chan struct{})

	// stream rule (we've not got audio, but use stream for more thorough test)
	streamRule := agg.Rule{Stream: "stream/large", Feeds: []string{"video0", "audio"}}

	// destination rules
	ID := "00" //same for both as changing, not duplicating, destination

	// bearer tokens for the connections
	var claims permission.Token

	claims.IssuedAt = time.Now().Unix() - 1
	claims.NotBefore = time.Now().Unix() - 1
	claims.ExpiresAt = time.Now().Unix() + 30
	claims.Audience = audience
	claims.Topic = "123"
	claims.ConnectionType = "session"
	claims.Scopes = []string{"write"}

	token0 := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign and get the complete encoded token as a string using the secret
	bearer0, err := token0.SignedString([]byte(secret))
	assert.NoError(t, err)

	// Make token for different topic
	claims.Topic = "456"
	token1 := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	bearer1, err := token1.SignedString([]byte(secret))
	assert.NoError(t, err)

	destination0 := audience + "/session/123"
	destination1 := audience + "/session/456"

	destinationRule0 := rwc.Rule{Stream: "stream/large", Destination: destination0, ID: ID, Token: bearer0}
	destinationRule1 := rwc.Rule{Stream: "stream/large", Destination: destination1, ID: ID, Token: bearer1}

	// set up some receivers for the stream

	ctx, cancel := context.WithCancel(context.Background())

	claims.Topic = "123"
	claims.Scopes = []string{"read"}
	token0read := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	bearer0read, err := token0read.SignedString([]byte(secret))
	assert.NoError(t, err)
	claims.Topic = "456"
	token1read := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	bearer1read, err := token1read.SignedString([]byte(secret))
	assert.NoError(t, err)

	s0 := reconws.New()
	go s0.ReconnectAuth(ctx, destination0, bearer0read)

	s1 := reconws.New()
	go s1.ReconnectAuth(ctx, destination1, bearer1read)

	// receivers
	rxCount0 := 0
	rxCount1 := 0

	wg.Add(2)

	go func() {
		defer wg.Done()
		select {
		case <-s0.In:
			close(ffmpegRunning) //signal to main body of test
			rxCount0++
		case <-time.After(5 * time.Second):
			t.Error("Timeout on destination websocket server 0")
			return
		}
		for {
			select {
			case <-closed:
				return
			case msg, ok := <-s0.In:
				if !ok {
					t.Log("s0.In channel closed")
					return
				}
				if len(msg.Data) > 0 {
					rxCount0++
				}
			}
		}
	}()
	go func() {
		defer wg.Done()
		for {
			select {
			case msg, ok := <-s1.In:
				if !ok {
					t.Log("s1.In channel closed")
					return
				}
				if len(msg.Data) > 0 {
					rxCount1++
				}
			case <-time.After(5 * time.Second):
				t.Error("Timeout on destination websocket server 0")
				return
			case <-closed:
				return
			}
		}
	}()

	app.Hub.Add <- streamRule
	app.Websocket.Add <- destinationRule0

	time.Sleep(1 * time.Millisecond)

	// start ffmpeg; use default vw port
	dest := "http://localhost:8888/ts/video0"
	args := fmt.Sprintf("-re -f concat -i list.txt -f mpegts -codec:v mpeg1video -s 640x480 -b:v 1000k -r 24 -bf 0 %s", dest)
	argSlice := strings.Split(args, " ")
	cmd := exec.Command("ffmpeg", argSlice...)
	fp, err := filepath.Abs("./test")
	assert.NoError(t, err)
	cmd.Dir = fp

	go func() {
		err := cmd.Run()
		if err != nil {
			t.Error("ffmpeg", err)
		}
	}()

	select {
	case <-ffmpegRunning:
	case <-time.After(5 * time.Second): //avoid hanging if test failed
		t.Error("Timeout: ffmpeg too slow to send first frame / ffmpeg frames not received")
	}

	// wait five frames
	time.Sleep(5 * 42 * time.Millisecond)

	app.Websocket.Add <- destinationRule1

	// wait another 40 frames
	time.Sleep(40 * 42 * time.Millisecond)

	if rxCount0 < 5 {
		t.Errorf("Insufficient frames received by server 0: %d", rxCount0)
	}

	if rxCount1 < 5 {
		t.Errorf("Insufficient frames received by server 1: %d", rxCount1)
	}

	cancel()
	close(app.Closed)
	close(closed)
	wg.Wait()
}

func reportSize(w http.ResponseWriter, r *http.Request, msgSize chan int) {
	c, err := testUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		msgSize <- len(message)
	}
}
