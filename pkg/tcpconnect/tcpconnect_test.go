package tcpconnect

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
)

func TestDialEcho(t *testing.T) {

	//log.SetLevel(log.TraceLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	port, err := freeport.GetFreePort()
	if err != nil {
		t.Fatal(err)
	}

	uri := ":" + strconv.Itoa(port)

	echo := New()
	go echo.Echo(ctx, uri)

	c := New()
	go c.Dial(ctx, "localhost"+uri)

	data := []byte("ping")

	timeout := 100 * time.Millisecond

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
