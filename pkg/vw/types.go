package vw

import (
	"bytes"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/timdrysdale/relay/pkg/agg"
	"github.com/timdrysdale/relay/pkg/hub"
	"github.com/timdrysdale/relay/pkg/rwc"
)

type App struct {
	Closed    chan struct{}
	Hub       *agg.Hub
	Opts      Specification
	Websocket *rwc.Hub
	WaitGroup sync.WaitGroup
}

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

type Packet struct {
	Data []byte
}

type Specification struct {
	Port               int    `default:"8888"`
	LogLevel           string `split_words:"true" default:"PANIC"`
	MuxBufferLength    int    `default:"10"`
	ClientBufferLength int    `default:"5"`
	ClientTimeoutMs    int    `default:"1000"`
	HttpWaitMs         int    `default:"5000"`
	HttpFlushMs        int    `default:"5"`
	HttpTimeoutMs      int    `default:"1000"`
	CpuProfile         string `default:""`
	API                string `default:""`
}
