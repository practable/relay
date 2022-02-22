package vw

import (
	"bytes"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/timdrysdale/relay/pkg/agg"
	"github.com/timdrysdale/relay/pkg/hub"
	"github.com/timdrysdale/relay/pkg/rwc"
)

// App represents a vw instance
type App struct {
	Closed    chan struct{}
	Hub       *agg.Hub
	Opts      Specification
	Websocket *rwc.Hub
	WaitGroup sync.WaitGroup
}

// WsHandlerClient represents a client handling an incoming websocket connection
type WsHandlerClient struct {
	Messages   *hub.Client
	Conn       *websocket.Conn
	UserAgent  string //r.UserAgent()
	RemoteAddr string //r.Header.Get("X-Forwarded-For")
}

type mutexBuffer struct {
	mux sync.Mutex
	b   bytes.Buffer
}

// Packet represents data
type Packet struct {
	Data []byte
}

// Specification represents key parameters for the vw instance
type Specification struct {
	Port               int    `default:"8888"`
	LogLevel           string `split_words:"true" default:"PANIC"`
	MuxBufferLength    int    `default:"10"`
	ClientBufferLength int    `default:"5"`
	ClientTimeoutMs    int    `default:"1000"`
	HTTPWaitMs         int    `default:"5000"`
	HTTPFlushMs        int    `default:"5"`
	HTTPTimeoutMs      int    `default:"1000"`
	CPUProfile         string `default:""`
	API                string `default:""`
}
