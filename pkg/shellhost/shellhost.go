// Package shellhost manages connections between
//  a shellrelay and a host machine accepting login shell
// connections, such as sshd. Such protocols are server speak first
// so the relay alerts shellhost when a new client has connected
// to the relay. Shellhost then makes a new dediated connection
// to the login shell port, and the relay.
package shellhost

import (
	"context"
	"encoding/json"

	"net/url"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/relay/pkg/reconws"
	"github.com/timdrysdale/relay/pkg/shellbar"
	"github.com/timdrysdale/relay/pkg/tcpconnect"
)

// Shellhost connects to remote relay, and makes a new connection
// to local (localhost:{port}) every time it is alerted to a new
// connection by a shellbar.ConnectionAction
func Shellhost(ctx context.Context, local, remote, token string) {

	manager := reconws.New()
	go manager.ReconnectAuth(ctx, remote, token)

	connections := make(map[string]context.CancelFunc)
	var ca shellbar.ConnectionAction
	log.Info("shellhost starting")
	for {
		select {
		case <-ctx.Done():
			log.Trace("parent context cancelled; shellhost shutting down")
			for _, cancel := range connections {
				cancel()
			}
			log.Debug("shellhost done")
			return

		case msg, ok := <-manager.In:
			log.WithField("msg", string(msg.Data)).Trace("shellhost received manage message")
			if !ok {
				log.Fatal("manager.In closed unexpectedly; shellhost shutting down")
				for _, cancel := range connections {
					cancel()
				}
				log.Debug("shellhost done")
				return
			}

			err := json.Unmarshal(msg.Data, &ca)
			if err != nil {
				continue
			}

			_, err = url.ParseRequestURI(ca.URI)
			if err != nil {
				continue
			}

			switch ca.Action {
			case "connect":
				uCtx, uCancel := context.WithCancel(ctx)
				connections[ca.URI] = uCancel
				log.WithField("uri", ca.URI).Debug("shellhost started new connection")
				go newConnection(uCtx, local, ca.URI)

			case "disconnect":
				log.WithField("uri", ca.URI).Debug("shellhost disconnecting")
				uCancel, ok := connections[ca.URI]
				if !ok {
					continue
				}
				uCancel()
			}
		}
	}
}

func newConnection(ctx context.Context, local, remote string) {

	timeout := 1 * time.Second

	unique := reconws.New()
	go unique.Dial(ctx, remote)

	shell := tcpconnect.New()
	go shell.Dial(ctx, local)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case data, ok := <-shell.In:
				if !ok {
					return
				}
				select {
				case unique.Out <- reconws.WsMessage{Data: data, Type: websocket.BinaryMessage}:
				case <-time.After(timeout):
					log.Error("timeout sending message ")
					return
				}
			}

		}
	}()

	go func() {
		log.WithField("uri", local).Trace("newConnection is listening to relay")
		for {
			log.WithField("uri", local).Trace("newConnection is ctx.Done()")
			select {

			case <-ctx.Done():
				return
			case msg, ok := <-unique.In:
				log.WithFields(log.Fields{"uri": local, "len": len(msg.Data)}).Trace("newConnection got message from relay")
				if !ok {
					return
				}
				select {
				case shell.Out <- msg.Data:
				case <-time.After(timeout):

					log.Error("timeout sending message ")
					return
				}
			}

		}
	}()

	<-ctx.Done()
	log.WithField("uri", local).Trace("newConnection is done")
}
