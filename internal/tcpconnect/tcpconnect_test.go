package tcpconnect

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/phayes/freeport"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	debug := false
	if debug {
		log.SetReportCaller(true)
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&log.TextFormatter{FullTimestamp: false, DisableColors: true})
		defer log.SetOutput(os.Stdout)

	} else {
		var ignore bytes.Buffer
		logignore := bufio.NewWriter(&ignore)
		log.SetOutput(logignore)
	}
}

func TestDialEcho(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	port, err := freeport.GetFreePort()
	if err != nil {
		t.Fatal(err)
	}

	uri := ":" + strconv.Itoa(port)

	echo := New()
	go echo.Echo(ctx, uri)

	timeout := 100 * time.Millisecond
	time.Sleep(timeout)

	c := New()
	go c.Dial(ctx, "localhost"+uri)

	data := []byte("ping")

	select {
	case c.Out <- data:
	case <-time.After(timeout):
		t.Fatal("timeout sending message ")
	}

	select {
	case msg := <-c.In:
		assert.Equal(t, data, msg)
	case <-time.After(timeout):
		t.Fatal("timeout waiting message echo")
	}

}

func TestListenEchoHandler(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	port, err := freeport.GetFreePort()
	if err != nil {
		t.Fatal(err)
	}

	uri := ":" + strconv.Itoa(port)

	listener := New()
	go listener.Listen(ctx, uri, EchoHandler)

	timeout := 100 * time.Millisecond
	time.Sleep(timeout)

	c := New()
	go c.Dial(ctx, "localhost"+uri)

	data := []byte("ping")

	select {
	case c.Out <- data:
	case <-time.After(timeout):
		t.Fatal("timeout sending message ")
	}

	select {
	case msg := <-c.In:
		assert.Equal(t, data, msg)
	case <-time.After(timeout):
		t.Fatal("timeout waiting message echo")
	}

}
