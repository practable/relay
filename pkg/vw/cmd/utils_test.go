package cmd

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/timdrysdale/relay/pkg/agg"
	"github.com/timdrysdale/relay/pkg/rwc"
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

func writeDataFile(size int, name string) ([]byte, error) {

	data := make([]byte, size)
	rand.Read(data)

	err := ioutil.WriteFile(name, data, 0644)

	return data, err

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

func suppressLog() {
	var ignore bytes.Buffer
	logignore := bufio.NewWriter(&ignore)
	log.SetOutput(logignore)
}

func displayLog() {
	log.SetOutput(os.Stdout)
}
