package shellbar

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer (10MB)
	// Typical key frame at 640x480 is 60 * 188B ~= 11kB
	maxMessageSize = 1024 * 1024 * 10
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// TODO
// - per ip throttling with github.com/didip/tollbooth
// - package stats.html with
// - canary connection + reporting
// - add auth to stats connection
// - tidy comments to look better in docs
// - remove stale code and comments
// - redo load testing / performance / memory leakage
// - add benchmarking

// 4096 Bytes is the approx average message size
// this number does not limit message size
// So for key frames we just make a few more syscalls
// null subprotocol required by Chrome
// TODO restrict CheckOrigin
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	Subprotocols:    []string{"null"},
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func fpsFromNs(ns float64) float64 {
	return 1 / (ns * 1e-9)
}

func handleConnections(closed <-chan struct{}, parentwg *sync.WaitGroup, clientActionsChan chan clientAction, messagesFromMe chan message, config Config) {
	hub := newHub()
	go hub.run()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveWs(closed, hub, w, r, config)
	})

	var wg sync.WaitGroup
	wg.Add(1)

	go statsClient(closed, &wg, hub, config)

	addr := ":" + strconv.Itoa(config.Listen)

	h := &http.Server{Addr: addr, Handler: nil}

	go func() {
		if err := h.ListenAndServe(); err != nil {
			log.Info("ListenAndServe: ", err) //TODO upgrade to fatal once httptest is supported
		}
	}()

	<-closed

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h.Shutdown(ctx)
	wg.Wait()
	parentwg.Done()
	log.Trace("handleConnections is done")
}
