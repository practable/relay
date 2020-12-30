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

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/relay/pkg/reconws"
	"github.com/timdrysdale/relay/pkg/shellbar"
	"github.com/timdrysdale/relay/pkg/tcpconnect"
)

// Shellhost connects to remote relay, and makes a new connection
// to local (localhost:{port}) every time it is alerted to a new
// connection by a shellbar.ConnectionAction
func Host(ctx context.Context, local, remote, token string) {

	id := "shellhost.Shellhost()"

	log.WithFields(log.Fields{"local": local, "remote": remote}).Infof("%s: STARTING", id)

	manager := reconws.New()
	go manager.ReconnectAuth(ctx, remote, token)

	connections := make(map[string]context.CancelFunc)
	var ca shellbar.ConnectionAction
	log.Info("shellhost starting")
	for {
		select {
		case <-ctx.Done():
			log.Tracef("%s: about to shut down because context cancelled", id)
			for _, cancel := range connections {
				cancel()
			}
			log.Debugf("%s: shutdown because context cancelled", id)
			return

		case msg, ok := <-manager.In:
			log.WithField("msg", string(msg.Data)).Debugf("%s: message received on manager.In", id)

			if !ok {
				log.Fatalf("%s: channel manager.In closed unexpectedly - shutting down", id)
				for _, cancel := range connections {
					cancel()
				}
				log.Debugf("%s: shutdown because manager.In closed unexpectedly", id)
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
				connections[ca.UUID] = uCancel
				log.WithFields(log.Fields{"local": local, "uri": ca.URI, "uuid": ca.UUID}).Infof("%s: started new connection", id)
				go newConnection(uCtx, local, ca.URI)

			case "disconnect":
				uCancel, ok := connections[ca.UUID]
				if !ok {
					log.WithFields(log.Fields{"local": local, "uri": ca.URI, "uuid": ca.UUID}).Warnf("%s: can't stop non-existent connection; ignoring", id)
					continue
				}
				log.WithFields(log.Fields{"local": local, "uri": ca.URI, "uuid": ca.UUID}).Infof("%s: stopping existing connection", id)
				uCancel()
			}
		}
	}

}

func newConnection(ctx context.Context, local, remote string) {

	timeout := 1 * time.Second

	id := "shellhost.newConnection(" + uuid.New().String()[0:6] + ")"

	log.WithFields(log.Fields{"local": local, "remote": remote}).Infof("%s: STARTING", id)

	unique := reconws.New()
	go unique.Dial(ctx, remote)
	log.WithFields(log.Fields{"to": remote}).Debugf("%s: started unique connection to relay", id)

	shell := tcpconnect.New()
	go shell.Dial(ctx, local)
	log.WithFields(log.Fields{"to": local}).Debugf("%s: started unique connection to sshd", id)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case data, ok := <-shell.In:
				log.WithFields(log.Fields{"to": local, "from": remote, "size": len(data)}).Debugf("%s: GOT %d-byte message FROM SHELL", id, len(data))
				if !ok {
					return
				}
				select {
				case unique.Out <- reconws.WsMessage{Data: data, Type: websocket.BinaryMessage}:
					log.WithFields(log.Fields{"to": remote, "from": local, "size": len(data)}).Debugf("%s: SENT %d-byte message TO RELAY", id, len(data))
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
				log.WithFields(log.Fields{"to": local, "from": remote, "size": len(msg.Data)}).Debugf("%s: GOT %d-byte message FROM RELAY", id, len(msg.Data))
				if !ok {
					return
				}
				select {
				case shell.Out <- msg.Data:
					log.WithFields(log.Fields{"to": local, "from": remote, "size": len(msg.Data)}).Debugf("%s: SENT %d-byte Message TO SHELL", id, len(msg.Data))
				case <-time.After(timeout):

					log.Error("timeout sending message ")
					return
				}
			}

		}
	}()

	<-ctx.Done()
	log.WithFields(log.Fields{"local": local, "remote": remote}).Infof("%s: DONE", id)
}
