// Package shellclient provides a client which listens on a local tcp port and for each incoming
// connection makes a unique connection to a remote shellrelay
package shellclient

import (
	"context"
	"strconv"

	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/relay/pkg/reconws"
	"github.com/timdrysdale/relay/pkg/tcpconnect"
)

// Client runs a listener for local ssh connections on a local tcp port
// and for each incoming connection makes a unique connection to a remote shellrelay
// Run multiple instances to listen on multiple ports
func Client(ctx context.Context, listen int, remote, token string) {

	id := "shellclient.ShellClient()"

	log.WithFields(log.Fields{"listen at": listen, "remote": remote}).Infof("%s: starting to listen at %d", id, listen)

	uri := ":" + strconv.Itoa(listen)

	listener := tcpconnect.New()
	go listener.Listen(ctx, uri, relayHandlerWithArgs(remote, token))

	<-ctx.Done()
	log.Infof("%s: done listening at %d", id, listen)
}

func relayHandlerWithArgs(remote, token string) func(context.Context, *tcpconnect.TCPconnect) {

	return func(ctx context.Context, c *tcpconnect.TCPconnect) {

		timeout := time.Second

		id := "shellclient.relayHandler(" + c.ID + ")"

		go c.HandleConn(ctx, *c.Conn) // messages on c.In and c.Out

		r := reconws.New()

		go r.ReconnectAuth(ctx, remote, token)

		log.WithFields(log.Fields{"r.ID": r.ID, "remote": remote}).Debugf("%s: started reconws(%s) to connect to %s", id, r.ID, remote)

		go func(ctx context.Context, tc *tcpconnect.TCPconnect) {

			for {
				select {
				case <-ctx.Done():
					log.Infof("%s: context cancelled; done", id)
					return
				case data, ok := <-tc.In:
					if !ok {
						log.Debugf("%s: local channel error, closing local read pump", id)
						return
					}
					size := len(data)
					select {
					case r.Out <- reconws.WsMessage{Data: data, Type: websocket.BinaryMessage}:
						log.WithField("size", size).Debugf("%s: sent %d-bytes from local to relay", id, size)
					case <-time.After(timeout):
						log.WithField("size", size).Debugf("%s: timeout waiting to send %d-bytes from local to local", id, size)
					}

				}

			}

		}(ctx, c)

		go func(ctx context.Context, tc *tcpconnect.TCPconnect) {

			for {
				select {
				case <-ctx.Done():
					log.Infof("%s: context cancelled; done", id)
					return
				case msg, ok := <-r.In:
					if !ok {
						log.Debugf("%s: relay channel error, closing relay read pump", id)
						return
					}
					size := len(msg.Data)
					select {
					case tc.Out <- msg.Data:
						log.WithField("size", size).Debugf("%s: sent %d-bytes from relay to local", id, size)
					case <-time.After(timeout):
						log.WithField("size", size).Debugf("%s: timeout waiting to send %d-bytes from relay to local", id, size)
					}
				}
			}

		}(ctx, c)

		<-ctx.Done()
		log.Infof("%s: done", id)
	}
}
