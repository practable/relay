package cmd

import (
	"bufio"
	"io"
	"net/http"
	"time"

	"github.com/gobwas/ws"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/relay/pkg/hub"
)

func (app *App) handleTs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	topic := vars["feed"]

	name := uuid.New().String()[:3]
	myDetails := &hub.Client{Hub: app.Hub.Hub,
		Name:  name,
		Send:  make(chan hub.Message),
		Stats: hub.NewClientStats(),
		Topic: topic}

	//receive MPEGTS in 188 byte chunks
	//ffmpeg uses one tcp packet per frame

	maxFrameBytes := 1024000 //TODO make configurable

	var frameBuffer mutexBuffer

	rawFrame := make([]byte, maxFrameBytes)

	glob := make([]byte, maxFrameBytes)

	frameBuffer.b.Reset() //else we send whole buffer on first flush

	reader := bufio.NewReader(r.Body)

	tCh := make(chan int)

	//Method for detecting packet boundaries: identify empty buffer via delay on reading a byte
	//after 13.738µs got 188 bytes
	//after 13.027µs got 120 bytes
	//after 13.883µs got 68 bytes
	//after 9.027µs got 188 bytes
	//after 8.876µs got 188 bytes
	//after 9.027µs got 104 bytes
	//<ffmpeg frame reported>
	//after 42.418638ms got 84 bytes  <============= NOTE THE ~40ms delay=====================
	//after 87.442µs got 188 bytes
	//after 43.555µs got 167 bytes
	//after 44.251µs got 21 bytes
	//after 23.267µs got 101 bytes
	//after 23.976µs got 49 bytes

	// drain but ignore messages from the hub
	go func() {
		for {
			select {
			case <-myDetails.Send:
			case <-app.Closed:
			}
		}
	}()

	// Read from the buffer, blocking if empty
	go func() {

		for {

			tCh <- 0 //tell the monitoring routine we're alive

			n, err := io.ReadAtLeast(reader, glob, 1)

			if err == nil {

				frameBuffer.mux.Lock()

				_, err = frameBuffer.b.Write(glob[:n])

				frameBuffer.mux.Unlock()

				if err != nil {
					log.Errorf("%v", err) //was Fatal?
					return
				}

			} else {

				return // avoid spinning our wheels

			}
		}
	}()

	for {

		select {

		case <-tCh:

			// do nothing, just received data from buffer

		case <-time.After(1 * time.Millisecond):
			// no new data for >= 1mS weakly implies frame has been fully sent to us
			// this is two orders of magnitude more delay than when reading from
			// non-empty buffer so _should_ be ok, but recheck if errors crop up on
			// lower powered system. Assume am on same computer as capture routine

			//flush buffer to internal send channel
			frameBuffer.mux.Lock()

			n, err := frameBuffer.b.Read(rawFrame)

			frame := rawFrame[:n]

			frameBuffer.b.Reset()

			frameBuffer.mux.Unlock()

			if err == nil && n > 0 {
				msg := hub.Message{Sender: *myDetails, Type: int(ws.OpBinary), Data: frame, Sent: time.Now()}
				app.Hub.Broadcast <- msg
			}

		case <-app.Closed:
			log.WithFields(log.Fields{"Name": name, "Topic": topic}).Info("http.muxHandler closed")
			return
		}
	}
}
