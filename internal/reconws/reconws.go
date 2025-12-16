/*
   reconws is websocket client that automatically reconnects
   Copyright (C) 2019 Timothy Drysdale <timothy.d.drysdale@gmail.com>

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as
   published by the Free Software Foundation, either version 3 of the
   License, or (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package reconws

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jpillora/backoff"
	"github.com/practable/relay/internal/access/restapi/operations"
	log "github.com/sirupsen/logrus"
)

// WsMessage represents a websocket message
type WsMessage struct {
	Data []byte
	Type int
	To   string
	From string
}

// ReconWs represents a websocket client that will reconnect if the connection is closed
// connects (retrying/reconnecting if necessary) to websocket server at url
type ReconWs struct {
	Connected       chan struct{} // allow notification of successful connection, helps with testing
	ConnectedAt     time.Time
	ForwardIncoming bool
	In              chan WsMessage
	Out             chan WsMessage
	Retry           RetryConfig
	URL             string
	ID              string
}

// RetryConfig represents the parameters for when to retry to connect
type RetryConfig struct {
	Factor  float64
	Jitter  bool
	Min     time.Duration
	Max     time.Duration
	Timeout time.Duration
}

// New returns a pointer to a new reconnecting websocket client ReconWs
func New() *ReconWs {
	r := &ReconWs{
		Connected: make(chan struct{}),
		// don't initialise connectedAt; set when connected
		In:              make(chan WsMessage),
		Out:             make(chan WsMessage),
		ForwardIncoming: true,
		Retry: RetryConfig{Factor: 2,
			Min:     1 * time.Second,
			Max:     10 * time.Second,
			Timeout: 1 * time.Second,
			Jitter:  false},
		ID: uuid.New().String()[0:6],
	}
	return r
}

// Reconnect sets URL to connect to, and runs the client
// run this in a separate goroutine so that the connection can be
// ended from where it was initialised, by close((* ReconWs).Stop)
// does not implement the connected signal; use ReconnectAuth for that
func (r *ReconWs) Reconnect(ctx context.Context, url string) {

	id := "reconws.ReconnectAuth(" + r.ID + ")"

	boff := &backoff.Backoff{
		Min:    r.Retry.Min,
		Max:    r.Retry.Max,
		Factor: r.Retry.Factor,
		Jitter: r.Retry.Jitter,
	}

	// try dialling ....

	for {

		select {
		case <-ctx.Done():
			return
		default:

			dialCtx, cancel := context.WithCancel(ctx)

			err := r.Dial(dialCtx, url)
			cancel()

			log.WithField("error", err).Debug("Dial finished")
			if err == nil {
				boff.Reset()
				log.Tracef("%s: dial finished successfully, resetting timeout to zero", id)
			} else {
				time.Sleep(boff.Duration())
				log.WithField("error", err).Tracef("%s: Dial finished with error, increasing timeout", id)
			}
			//TODO immediate return if cancelled....
		}
	}
}

// ReconnectAuth reconnects to a relay instance that uses an access server
// to gatekeep access to the websocket relay
// run this in a separate goroutine so that the connection can be
// ended from where it was initialised, by close((* ReconWs).Stop)
func (r *ReconWs) ReconnectAuth(ctx context.Context, url, token string) {

	id := "reconws.ReconnectAuth(" + r.ID + ")"

	boff := &backoff.Backoff{
		Min:    r.Retry.Min,
		Max:    r.Retry.Max,
		Factor: r.Retry.Factor,
		Jitter: r.Retry.Jitter,
	}

	// try dialling ....

	waitBeforeDial := false

	for {

		select {
		case <-ctx.Done():
			return
		default:

			if waitBeforeDial {
				time.Sleep(boff.Duration())
			}

			waitBeforeDial = true

			var client = &http.Client{
				Timeout: time.Second * 10,
			}

			req, err := http.NewRequest("POST", url, nil)
			if err != nil {
				log.WithField("error", err).Warnf("%s: failed to create request", id)
				continue
			}

			req.Header.Add("Authorization", token)

			resp, err := client.Do(req)

			if err != nil {
				log.WithField("error", err).Warnf("%s: failed request to access endpoint", id)
				continue
			}

			body, err := io.ReadAll(resp.Body)

			if err != nil {
				log.WithField("error", err).Warnf("%s: failed reading access response body", id)
				continue
			}

			var session operations.SessionOKBody

			err = json.Unmarshal(body, &session)

			if err != nil {

				log.WithFields(log.Fields{"error": err, "body": string(body)}).Debugf("%s: failed marshalling access response into struct", id)
				continue
			}

			log.WithFields(log.Fields{"body": session, "URI": session.URI}).Infof("%s: successful relay access request", id)

			dialCtx, cancel := context.WithCancel(ctx)

			err = r.Dial(dialCtx, session.URI)
			cancel()

			if err == nil {
				boff.Reset()
				waitBeforeDial = false
				log.Tracef("%s: dial finished successfully, resetting timeout to zero", id)
			} else {
				log.WithField("error", err).Tracef("%s: Dial finished with error, increasing timeout", id)
			}
			//TODO immediate return if cancelled....?
		}
	}
}

// Dial the websocket server once.
// If dial fails then return immediately
// If dial succeeds then handle message traffic until
// the context is cancelled
func (r *ReconWs) Dial(ctx context.Context, urlStr string) error {

	id := "reconws.Dial(" + r.ID + ")"

	var err error

	if urlStr == "" {
		log.Errorf("%s: Can't dial an empty Url", id)
		return errors.New("Can't dial an empty Url")
	}

	// parse to check, dial with original string
	u, err := url.Parse(urlStr)

	if err != nil {
		log.Errorf("%s: error with url because %s:", id, err.Error())
		return err
	}

	if u.Scheme != "ws" && u.Scheme != "wss" {
		log.Errorf("%s: Url needs to start with ws or wss", id)
		return errors.New("Url needs to start with ws or wss")
	}

	if u.User != nil {
		log.Errorf("%s: Url can't contain user name and password", id)
		return errors.New("Url can't contain user name and password")
	}

	// start dialing ....

	log.WithField("To", u).Tracef("%s: connecting to %s", id, u)

	//assume our context has been given a deadline if needed
	c, _, err := websocket.DefaultDialer.DialContext(ctx, urlStr, nil)
	//	defer c.Close()

	if err != nil {
		log.WithField("error", err).Errorf("%s: dialing error because %s", id, err.Error())
		return err
	}

	// assume we are connected?
	r.ConnectedAt = time.Now()
	close(r.Connected) //signal that we've connected
	defer func() {
		r.Connected = make(chan struct{}) //reset for next time
	}()

	log.WithField("To", u).Tracef("%s: connected to %s", id, u)
	// handle our reading tasks

	readClosed := make(chan struct{})

	go func() {
	LOOP:
		for {
			select {
			case <-ctx.Done():
			default:
			}
			//assume this will produce non-nil err on context.Done
			mt, data, err := c.ReadMessage()

			// Check for errors, e.g. caused by writing task closing conn
			// because we've been instructed to exit
			// log as info since we expect an error here on a normal exit
			if err != nil {
				log.WithField("error", err).Infof("%s: error reading from conn; closing", id)
				close(readClosed)
				break LOOP
			}

			// optionally forward messages
			if r.ForwardIncoming {
				r.In <- WsMessage{Data: data, Type: mt}
				log.Tracef("%s: received %d-byte message", id, len(data))

			} else {
				log.Tracef("%s: ignored %d-byte message", id, len(data))
			}
		}
	}()

	// handle our writing tasks
LOOPWRITING:
	for {
		select {
		case <-readClosed:
			err = nil // nil error resets the backoff
			break LOOPWRITING
		case msg := <-r.Out:

			err := c.WriteMessage(msg.Type, msg.Data)
			if err != nil {
				log.WithField("error", err).Infof("%s: error writing to conn; closing", id)
				break LOOPWRITING
			}
			log.Tracef("%s: sent %d-byte message", id, len(msg.Data))

		case <-ctx.Done(): // context has finished, either timeout or cancel
			//TODO - do we need to do this?
			// Cleanly close the connection by sending a close message
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.WithField("error", err).Infof("%s: error sending close message; closing", id)
			} else {
				log.Infof("%s: connection closed", id)
			}
			c.Close()
			break LOOPWRITING
		}
	}
	log.Tracef("%s: done", id)
	return err

}
