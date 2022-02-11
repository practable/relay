// Package tcpconnect provides a tcp dialer
// with a two-channel []byte{} interface (In, Out)
package tcpconnect

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// Dial the local tcp address once.
// If dial fails then return immediately
// If dial succeeds then handle message traffic until
// the context is cancelled

type mutexBuffer struct {
	mux sync.Mutex
	b   bytes.Buffer
}

// TCPconnect represents a TCP connection
type TCPconnect struct {
	// In channel carries messages FROM TCP conn
	In chan []byte

	//Out channel carries messages TO TCP conn
	Out chan []byte

	// MaxFrameBytes represents the maximum size of a single message
	MaxFrameBytes int

	// Listeners is a list of connections when in listening mode
	Listeners []*TCPconnect

	// ID identifies the instance, useful for logging
	ID string

	// Conn holds a pointer to the net.conn
	Conn *net.Conn
}

// New returns a pointer to new TCPconnect struct
func New() *TCPconnect {
	return &TCPconnect{
		In:            make(chan []byte),
		Out:           make(chan []byte),
		MaxFrameBytes: 1024 * 1024,
		ID:            uuid.New().String()[0:6],
	}
}

// WithMaxFrameBytes sets the maximum bytes that can be sent per frame
// this impacts memory usage (larger max frame sizes require more memory for the buffer)
func (c *TCPconnect) WithMaxFrameBytes(max int) *TCPconnect {
	c.MaxFrameBytes = max
	return c
}

// WithConn sets which net.Conn is used
func (c *TCPconnect) WithConn(conn *net.Conn) *TCPconnect {
	c.Conn = conn
	return c
}

// Dial starts a connection to the given URI
func (c *TCPconnect) Dial(ctx context.Context, uri string) {

	id := "tcpconnect.Dial(" + c.ID + ")"

	var err error

	if uri == "" {
		log.Errorf("%s: Can't dial an empty Url", id)
		return
	}

	// parse to check, dial with original string
	_, err = url.Parse(uri)

	if err != nil {
		log.Errorf("%s: error with url %s of %s", id, uri, err.Error())
		return
	}

	// start dialing ....

	log.WithField("To", uri).Tracef("%s: connecting to %s", id, uri)

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", uri)

	if err != nil {
		log.WithFields(log.Fields{"uri": uri, "error": err.Error()}).Errorf("%s: failed to dial %s because %s", id, uri, err.Error())
		return
	}

	defer conn.Close()

	log.WithField("To", uri).Debugf("%s: connected to %s", id, uri)

	c.HandleConn(ctx, conn)
}

// Listen returns new conns over a channel for use by other handlers
func (c *TCPconnect) Listen(ctx context.Context, uri string, handler func(context.Context, *TCPconnect)) {

	id := "tcpconnect.Listen(" + c.ID + ")"

	lc := &net.ListenConfig{}

	l, err := lc.Listen(ctx, "tcp", uri)

	if err != nil {
		log.WithField("uri", uri).Debugf("%s: error connecting to %s because %s", id, uri, err.Error())
		return
	}

	defer l.Close()

	log.WithField("uri", uri).Debugf("%s: awaiting connections at %s", id, uri)

	for {

		// Wait for a connection.
		conn, err := l.Accept()

		if err != nil {
			log.WithFields(log.Fields{"uri": uri, "err": err.Error()}).Warnf("%s: failed to accept connection because %s", id, err.Error())
			continue
		}
		// Handle the connection in a new goroutine.
		// The loop then returns to accepting, so that
		// multiple connections may be served concurrently.
		listener := New().WithConn(&conn)
		go handler(ctx, listener)
		log.WithField("uri", uri).Debugf("%s: got a new connection (%s)", id, listener.ID)
	}

}

// SpeakThenEchoHandler is used for testing (it says something then echoes)
func SpeakThenEchoHandler(ctx context.Context, c *TCPconnect) {

	id := "tcpconnect.SpeakThenEchoHandler(" + c.ID + ")"

	go c.HandleConn(ctx, *c.Conn)

	go func(ctx context.Context, c *TCPconnect) {

		greeting := []byte("Echo Service")
		c.Out <- greeting
		log.WithFields(log.Fields{"size": len(greeting)}).Debugf("%s: spoke first with a %d-byte message", id, len(greeting))

		for {
			select {
			case <-ctx.Done():
			case msg, ok := <-c.In:
				if !ok {
					log.WithFields(log.Fields{"msg": string(msg)}).Debugf("%s: channel error, closing", id)
					return
				}
				c.Out <- msg
				log.WithFields(log.Fields{"msg": string(msg)}).Debugf("%s: echo'd a %d-byte message", id, len(msg))
			}

		}

	}(ctx, c)

}

// EchoHandler is for testing (it echoes messages)
func EchoHandler(ctx context.Context, c *TCPconnect) {

	id := "tcpconnect.EchoHandler(" + c.ID + ")"

	go c.HandleConn(ctx, *c.Conn)

	go func(ctx context.Context, c *TCPconnect) {

		for {
			select {
			case <-ctx.Done():
			case msg, ok := <-c.In:
				if !ok {
					log.WithFields(log.Fields{"msg": string(msg)}).Debugf("%s: channel error, closing", id)
					return
				}
				c.Out <- msg
				log.WithFields(log.Fields{"msg": string(msg)}).Debugf("%s: echo'd a %d-byte message", id, len(msg))
			}

		}

	}(ctx, c)

}

// Echo echoes messages (used for testing)
func (c *TCPconnect) Echo(ctx context.Context, uri string) {

	id := "tcpconnect.Echo(" + c.ID + ")"

	lc := &net.ListenConfig{}

	l, err := lc.Listen(ctx, "tcp", uri)

	if err != nil {
		log.WithFields(log.Fields{"uri": uri, "err": err.Error()}).Warnf("%s: failed to listener because %s", id, err.Error())
		return
	}

	defer l.Close()

	go func(l net.Listener) {

		log.WithField("uri", uri).Debugf("%s: awaiting connections at %s", id, uri)

		for {

			// Wait for a connection.
			conn, err := l.Accept()

			if err != nil {
				log.WithFields(log.Fields{"uri": uri, "err": err.Error()}).Warnf("%s: failed to accept connection because %s", id, err.Error())
				return //the context is probably cancelled.
			}
			// Handle the connection in a new goroutine.
			// The loop then returns to accepting, so that
			// multiple connections may be served concurrently.

			listener := New()
			c.Listeners = append(c.Listeners, listener)

			log.WithField("uri", uri).Debugf("%s: got a new connection (%s)", id, listener.ID)

			go listener.HandleConn(ctx, conn)

			go func(ctx context.Context, c *TCPconnect) {
				id := "tcpconnect.Echo.Connector(" + c.ID + ")"
				for {
					select {
					case <-ctx.Done():
					case msg, ok := <-listener.In:
						if !ok {
							log.WithFields(log.Fields{"uri": uri, "msg": string(msg)}).Debugf("%s: channel error, closing", id)
							return
						}
						listener.Out <- msg
						log.WithFields(log.Fields{"uri": uri, "msg": string(msg)}).Debugf("%s: echo'd a %d-byte message", id, len(msg))
					}

				}

			}(ctx, listener)
		}

	}(l)
	<-ctx.Done()

}

// HandleConn handles connections, including reading and writing
func (c *TCPconnect) HandleConn(ctx context.Context, conn net.Conn) {

	id := "tcpconnect.handleConn(" + c.ID + ")"

	var frameBuffer mutexBuffer

	rawFrame := make([]byte, c.MaxFrameBytes)

	glob := make([]byte, c.MaxFrameBytes)

	frameBuffer.b.Reset() //else we send whole buffer on first flush

	reader := bufio.NewReader(conn)

	tCh := make(chan int)

	// write messages to the destination
	go func() {
		for {
			select {
			case data := <-c.Out:
				n, err := conn.Write(data)

				if err != nil {
					log.Warnf("%s: error writing  %d-byte message to conn because %s", id, len(data), err.Error())
				}
				if n == len(data) {
					log.Warnf("%s: wrote %d of %d byte message to conn", id, n, len(data))
				}

				log.Debugf("%s: wrote %d-byte message to conn", id, len(data))
			case <-ctx.Done():
				log.Debugf("%s: write pump context cancelled", id)
				return
				//put this option here to avoid spinning our wheels
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
				log.Warnf("%s: error conn writing into frame buffer  %s", id, err.Error())
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
			// lower powered system.

			//flush buffer to internal send channel
			frameBuffer.mux.Lock()

			n, err := frameBuffer.b.Read(rawFrame)

			frame := rawFrame[:n]

			frameBuffer.b.Reset()

			frameBuffer.mux.Unlock()

			if err == nil && n > 0 {
				c.In <- frame
				log.Debugf("%s: wrote %d-byte message to channel", id, n)
			}
			// don't report errors - just an empty buffer which is not a problem....

		case <-ctx.Done():
			log.Debugf("%s: read pump context cancelled", id)
			return
		}
	}
}
