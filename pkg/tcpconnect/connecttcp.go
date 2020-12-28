// Package tcpconnect provides a tcp dialer
// with a two-channel []byte{} interface (In, Out)
package tcpconnect

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/url"
	"sync"
	"time"

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

type TCPconnect struct {
	In            chan []byte
	Out           chan []byte
	MaxFrameBytes int
}

func New() *TCPconnect {
	return &TCPconnect{
		In:            make(chan []byte),
		Out:           make(chan []byte),
		MaxFrameBytes: 1024 * 1024,
	}
}

func (c *TCPconnect) WithMaxFrameBytes(max int) *TCPconnect {
	c.MaxFrameBytes = max
	return c
}

func (c *TCPconnect) Dial(ctx context.Context, uri string) error {

	var err error

	if uri == "" {
		log.Error("Can't dial an empty Url")
		return errors.New("Can't dial an empty Url")
	}

	// parse to check, dial with original string
	u, err := url.Parse(uri)

	if err != nil {
		log.Error("Url:", err)
		return err
	}

	// start dialing ....

	log.WithField("To", u).Debug("Connecting")

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", uri)

	if err != nil {
		log.WithFields(log.Fields{"uri": uri, "error": err.Error()}).Error("Failed to dial")
		return err
	}

	defer conn.Close()

	log.WithField("To", u).Info("Connected")

	return c.handleConn(ctx, conn)
}

func (c *TCPconnect) handleConn(ctx context.Context, conn net.Conn) error {

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
				conn.Write(data)
			case <-ctx.Done():
				//put this option here to avoid spinning our wheels
			}
		}
	}()

	// Read from the buffer, blocking if empty
	go func() {

		for {

			tCh <- 0 //tell the monitoring routine we're alive

			n, err := io.ReadAtLeast(reader, glob, 1)
			log.WithField("Count", n).Trace("Read from buffer")
			if err == nil {

				frameBuffer.mux.Lock()

				_, err = frameBuffer.b.Write(glob[:n])

				frameBuffer.mux.Unlock()

				if err != nil {
					log.Errorf("%v", err) //was Fatal?
					return
				}

			} else {
				log.Error("Read/FrameBuffer issue")
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
			}

		case <-ctx.Done():
			log.Debug("Read Pump Done")
			return nil
		}
	}
}
