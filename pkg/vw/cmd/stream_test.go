package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/timdrysdale/relay/pkg/agg"
	crossbar "github.com/timdrysdale/relay/pkg/crossbar/cmd"
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
	wssUrl := fmt.Sprintf("ws://localhost:%s", ue.Port())
	destinationRule := rwc.Rule{Stream: "stream/large", Destination: wssUrl, Id: "00"}
	app.Websocket.Add <- destinationRule

	time.Sleep(1 * time.Millisecond)

	uv, _ := url.Parse(s.URL)
	dest := fmt.Sprintf("http://localhost:%s/ts/video0", uv.Port())
	//dest := "http://localhost:8888/ts/video"
	args := fmt.Sprintf("-re -i sample.ts -f mpegts -codec:v mpeg1video -s 640x480 -b:v 1000k -r 24 -bf 0 %s", dest)
	argSlice := strings.Split(args, " ")
	cmd := exec.Command("ffmpeg", argSlice...)
	err := cmd.Run()
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

	go streamCmd.Run(streamCmd, nil) //streamCmd will populate the global app

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
	Id := "00" //same for both as changing, not duplicating, destination

	url0, _ := url.Parse(serverExternal0.URL)
	wss0 := fmt.Sprintf("ws://localhost:%s", url0.Port())
	destinationRule0 := rwc.Rule{Stream: "stream/large", Destination: wss0, Id: Id}

	url1, _ := url.Parse(serverExternal1.URL)
	wss1 := fmt.Sprintf("ws://localhost:%s", url1.Port())
	destinationRule1 := rwc.Rule{Stream: "stream/large", Destination: wss1, Id: Id}

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

	go streamCmd.Run(streamCmd, nil) //streamCmd will populate the global app

	// destination websocket reporting channels
	msgSize0 := make(chan int)
	msgSize1 := make(chan int)
	authToken := "some.test.token"

	//destination websocket servers
	serverExternal0 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reportSizeAuth(w, r, msgSize0, authToken) }))
	defer serverExternal0.Close()

	serverExternal1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reportSizeAuth(w, r, msgSize1, authToken) }))
	defer serverExternal1.Close()

	// destination changeover signalling (to save guessing ffmpeg startup time)
	ffmpegRunning := make(chan struct{})

	// stream rule (we've not got audio, but use stream for more thorough test)
	streamRule := agg.Rule{Stream: "stream/large", Feeds: []string{"video0", "audio"}}

	// destination rules
	Id := "00" //same for both as changing, not duplicating, destination

	url0, _ := url.Parse(serverExternal0.URL)
	wss0 := fmt.Sprintf("ws://localhost:%s", url0.Port())
	destinationRule0 := rwc.Rule{Stream: "stream/large", Destination: wss0, Id: Id, Token: authToken}

	url1, _ := url.Parse(serverExternal1.URL)
	wss1 := fmt.Sprintf("ws://localhost:%s", url1.Port())
	destinationRule1 := rwc.Rule{Stream: "stream/large", Destination: wss1, Id: Id, Token: authToken}

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

func reportSizeAuth(w http.ResponseWriter, r *http.Request, msgSize chan int, authToken string) {
	c, err := testUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()

	mt, message, err := c.ReadMessage()
	if err != nil {
		return
	}

	reply := crossbar.AuthMessage{
		Authorised: false,
		Token:      string(message),
		Reason:     "Denied", //not an official message ...
	}

	if string(message) == authToken {
		reply = crossbar.AuthMessage{
			Authorised: true,
			Reason:     "ok",
		}
	}

	message, err = json.Marshal(&reply)

	err = c.WriteMessage(mt, message)
	if err != nil {

		return
	}

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		msgSize <- len(message)
	}
}
