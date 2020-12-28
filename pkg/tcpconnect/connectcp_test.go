package tcpconnect

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
)

func TestDial(t *testing.T) {
	// Listen on TCP port 2000 on all available unicast and
	// anycast IP addresses of the local system.

	//log.SetLevel(log.TraceLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	port, err := freeport.GetFreePort()
	if err != nil {
		t.Fatal(err)
	}

	uri := ":" + strconv.Itoa(port)

	lc := &net.ListenConfig{}

	l, err := lc.Listen(ctx, "tcp", uri)

	if err != nil {
		t.Fatal(err)
	}

	defer l.Close()

	hosts := []*TCPconnect{}

	go func(l net.Listener) {

		for {
			// Wait for a connection.
			conn, err := l.Accept()
			if err != nil {
				return
			}
			// Handle the connection in a new goroutine.
			// The loop then returns to accepting, so that
			// multiple connections may be served concurrently.
			c := New()
			hosts = append(hosts, c)
			go c.handleConn(ctx, conn)
			go func(ctx context.Context, c *TCPconnect) {

				for {
					select {
					case <-ctx.Done():
					case msg, ok := <-c.In:
						assert.True(t, ok)
						c.Out <- msg
					}

				}

			}(ctx, c)
		}

	}(l)

	c := New()

	go c.Dial(ctx, "localhost"+uri)

	fmt.Println(err)

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
