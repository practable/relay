package vw

import (
	"log"
	"testing"

	"github.com/practable/relay/internal/agg"
	"github.com/practable/relay/internal/rwc"
)

var unix = "foo\r\n"
var windows = "foo\n"
var cleaned = "foo"

func TestUtilsClean(t *testing.T) {
	if clean(unix) != clean(windows) {
		log.Fatalf("did not clean strings %v %v\n", []byte(unix), []byte(windows))
	}

	if clean(unix) != cleaned {
		log.Fatalf("clean(string) did not match a cleaned string %v %v\n", []byte(unix), []byte(cleaned))
	}
}

func testApp(running bool) *App {
	a := &App{Hub: agg.New(), Closed: make(chan struct{})}
	a.Websocket = rwc.New(a.Hub)
	if running {
		go a.Hub.Run(a.Closed)
		go a.Websocket.Run(a.Closed)
	}
	return a
}
