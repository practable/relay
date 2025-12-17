package vw

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/practable/relay/internal/hub"
)

func TestHandleTsFrameBoundaries(t *testing.T) {
	//
	// +----------+       +----------+        +---------+             +----------+
	// |          |       |          |        |         |             |          |
	// | ffmpeg   |       | tsHandler|        |  Agg    | crx.Send    |  crx     |
	// |          +------>+  (under  +------->+         +-------------> (hub     |
	// |          |       |   test)  |        |         |             |  test    |
	// |          |       |          |        |         |             |  client) |
	// +----------+       +----------+        +---------+             +----------+
	//
	// TEST HARNESS--><--ITEM UNDER TEST--><---TEST HARNESS---------------------->
	//
	//                                       --diagram created using asciiflow.com
	//
	// Explanation:
	// What's under test is the tsHandler's ability to detect frame boundaries due to the
	// pause between frames that are being posted from (presumably) ffmpeg
	// Obviously this makes the assumption that there is slack time between frames
	// Which _should_ hold for even single board computers and consumer bitrates/framerates
	//
	// Test harness comprises a syscall to ffmpeg to stream some frames and hub client
	// which checks for those messages to have the appropriate size
	//
	// This also checks that the /ts is stripped from the path correctly so that the video
	// is forwarded to the right topic

	// Test harness, receiving side (agg, and hub.Client)
	//app := App{Closed: make(chan struct{}), Hub: agg.New()}
	//go app.Hub.Run(app.Closed)

	app := testApp(true)

	time.Sleep(2 * time.Millisecond)

	crx := &hub.Client{Hub: app.Hub.Hub, Name: "rx", Topic: "video", Send: make(chan hub.Message)}
	app.Hub.Register <- crx

	time.Sleep(2 * time.Millisecond)

	// check hubstats to see if registered ok
	if len(app.Hub.Hub.Clients) != 1 {
		t.Errorf("Wrong number of clients registered to hub wanted/got %d/%d", 1, len(app.Hub.Hub.Clients))
	}

	// server to action the handler under test
	r := mux.NewRouter()
	r.HandleFunc("/ts/{feed}", http.HandlerFunc(app.handleTs))

	s := httptest.NewServer(r)
	defer s.Close()

	time.Sleep(2 * time.Millisecond)

	// taken from a known good version of the code when running on the sample video
	frameSizes := []int{15980,
		20116,
		17296,
		16544,
		18988,
		//15792,
	}

	// as of 2025, with ffmpeg version 4.4.2-0ubuntu0.22.04.1 - these are the frame sizes
	frameSizesNew := []int{15604,
		18988,
		16356,
		16168,
		17672,
	}

	go func() {
		// did frame sizes come through correctly?
		for i := 0; i < len(frameSizes); i++ {
			select {
			case <-time.After(200 * time.Millisecond):
				t.Errorf("timed out on frame  %d", i)
			case msg, ok := <-crx.Send:
				if ok {
					fs := len(msg.Data)
					if fs != frameSizes[i] && fs != frameSizesNew[i] {
						t.Errorf("Frame %d content size wrong; got/wanted %v/%v\n", i, len(msg.Data), frameSizes[i])
					}
				} else {
					t.Error("channel not ok") //this test seems sensitive to timing off the sleeps, registration delay?
				}
			}
		}
	}()

	time.Sleep(1 * time.Millisecond)

	u, _ := url.Parse(s.URL)

	dest := fmt.Sprintf("http://localhost:%s/ts/video", u.Port())

	args := fmt.Sprintf("-re -i sample.ts -f mpegts -codec:v mpeg1video -s 640x480 -b:v 1000k -r 24 -bf 0 %s", dest)

	argSlice := strings.Split(args, " ")

	cmd := exec.Command("ffmpeg", argSlice...)

	cmd.Dir = "./test"

	err := cmd.Run()

	if err != nil {
		t.Error("ffmpeg", err)
	}

	// hang on long enough for timeouts in the anonymous goroutine to trigger
	time.Sleep(300 * time.Millisecond)

	close(app.Closed)

	time.Sleep(time.Millisecond) //allow time for goroutines to end before starting a new http server
}
