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
	"math/rand"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jpillora/backoff"
	log "github.com/sirupsen/logrus"
	crossbar "github.com/timdrysdale/crossbar/cmd"
	"github.com/timdrysdale/relay/pkg/chanstats"
)

type WsMessage struct {
	Data []byte
	Type int
}

// connects (retrying/reconnecting if necessary) to websocket server at url

type ReconWs struct {
	ForwardIncoming bool
	In              chan WsMessage
	Out             chan WsMessage
	Retry           RetryConfig
	Stats           *chanstats.ChanStats
	Url             string
}

type RetryConfig struct {
	Factor  float64
	Jitter  bool
	Min     time.Duration
	Max     time.Duration
	Timeout time.Duration
}

func New() *ReconWs {
	r := &ReconWs{
		In:              make(chan WsMessage),
		Out:             make(chan WsMessage),
		ForwardIncoming: true,
		Retry: RetryConfig{Factor: 2,
			Min:     1 * time.Second,
			Max:     10 * time.Second,
			Timeout: 1 * time.Second,
			Jitter:  false},
		Stats: chanstats.New(),
	}
	return r
}

// run this in a separate goroutine so that the connection can be
// ended from where it was initialised, by close((* ReconWs).Stop)
func (r *ReconWs) ReconnectAuth(ctx context.Context, url string, token string) {

	boff := &backoff.Backoff{
		Min:    r.Retry.Min,
		Max:    r.Retry.Max,
		Factor: r.Retry.Factor,
		Jitter: r.Retry.Jitter,
	}

	rand.Seed(time.Now().UTC().UnixNano())

	// try dialling ....

	for {

		select {
		case <-ctx.Done():
			return
		default:

			dialCtx, cancel := context.WithCancel(ctx)
			//defer cancel()
			err := r.DialAuth(dialCtx, url, token)
			cancel()

			log.WithField("error", err).Debug("Dial finished")
			if err == nil {
				boff.Reset()
			} else {
				time.Sleep(boff.Duration())
			}
			//TODO immediate return if cancelled....
		}
	}
}

// run this in a separate goroutine so that the connection can be
// ended from where it was initialised, by close((* ReconWs).Stop)
func (r *ReconWs) Reconnect(ctx context.Context, url string) {

	boff := &backoff.Backoff{
		Min:    r.Retry.Min,
		Max:    r.Retry.Max,
		Factor: r.Retry.Factor,
		Jitter: r.Retry.Jitter,
	}

	rand.Seed(time.Now().UTC().UnixNano())

	// try dialling ....

	for {

		select {
		case <-ctx.Done():
			return
		default:

			dialCtx, cancel := context.WithCancel(ctx)
			//defer cancel()
			err := r.Dial(dialCtx, url)
			cancel()

			log.WithField("error", err).Debug("Dial finished")
			if err == nil {
				boff.Reset()
			} else {
				time.Sleep(boff.Duration())
			}
			//TODO immediate return if cancelled....
		}
	}
}

// Dial the websocket server once, and submit auth token
// If dial fails then return immediately
// If dial succeeds then handle message traffic until
// the context is cancelled
func (r *ReconWs) DialAuth(ctx context.Context, urlStr string, token string) error {

	var err error

	if urlStr == "" {
		log.Error("Can't dial an empty Url")
		return errors.New("Can't dial an empty Url")
	}

	// parse to check, dial with original string
	u, err := url.Parse(urlStr)

	if err != nil {
		log.Error("Url:", err)
		return err
	}

	if u.Scheme != "ws" && u.Scheme != "wss" {
		log.Error("Url needs to start with ws or wss")
		return errors.New("Url needs to start with ws or wss")
	}

	if u.User != nil {
		log.Error("Url can't contain user name and password")
		return errors.New("Url can't contain user name and password")
	}

	// start dialing ....

	log.WithField("To", u).Debug("Connecting")

	//assume our context has been given a deadline if needed
	c, _, err := websocket.DefaultDialer.DialContext(ctx, urlStr, nil)
	//	defer c.Close()

	if err != nil {
		log.WithField("error", err.Error()).Error("Dialing")
		return err
	}

	// assume we are connected, so try auth ...

	err = c.WriteMessage(websocket.TextMessage, []byte(token))

	if err != nil {

		log.WithField("error", err.Error()).Error("Writing token for authorisation")
		return err
	}

	mt, data, err := c.ReadMessage()

	if mt == websocket.TextMessage {

		authMsg := &crossbar.AuthMessage{}

		json.Unmarshal(data, authMsg)

		if !authMsg.Authorised {

			reason := authMsg.Reason
			if reason == "" {
				reason = string(data)
			}

			return errors.New(reason)
		}
	} else {
		return errors.New("Auth reply format should be websocket.TextMessage but got websocket.Binary")
	}

	// auth successful ... so carry on
	r.Stats.ConnectedAt = time.Now()

	log.WithField("To", u).Info("Connected")

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
				log.WithField("info", err).Info("Reading")
				close(readClosed)
				break LOOP
			}
			// optionally forward messages
			if r.ForwardIncoming {
				r.In <- WsMessage{Data: data, Type: mt}
				//update stats
				r.Stats.Rx.Bytes.Add(float64(len(data)))
				r.Stats.Rx.Dt.Add(time.Since(r.Stats.Rx.Last).Seconds())
				r.Stats.Rx.Last = time.Now()
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
				log.WithField("error", err).Error("Writing")
				break LOOPWRITING
			}
			//update stats
			r.Stats.Tx.Bytes.Add(float64(len(msg.Data)))
			r.Stats.Tx.Dt.Add(time.Since(r.Stats.Tx.Last).Seconds())
			r.Stats.Tx.Last = time.Now()

		case <-ctx.Done(): // context has finished, either timeout or cancel
			//TODO - do we need to do this?
			// Cleanly close the connection by sending a close message
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.WithField("error", err).Error("Closing")
			} else {
				log.Info("Closed")
			}
			c.Close()
			break LOOPWRITING
		}
	}

	return err

}

// Dial the websocket server once.
// If dial fails then return immediately
// If dial succeeds then handle message traffic until
// the context is cancelled
func (r *ReconWs) Dial(ctx context.Context, urlStr string) error {

	var err error

	if urlStr == "" {
		log.Error("Can't dial an empty Url")
		return errors.New("Can't dial an empty Url")
	}

	// parse to check, dial with original string
	u, err := url.Parse(urlStr)

	if err != nil {
		log.Error("Url:", err)
		return err
	}

	if u.Scheme != "ws" && u.Scheme != "wss" {
		log.Error("Url needs to start with ws or wss")
		return errors.New("Url needs to start with ws or wss")
	}

	if u.User != nil {
		log.Error("Url can't contain user name and password")
		return errors.New("Url can't contain user name and password")
	}

	// start dialing ....

	log.WithField("To", u).Debug("Connecting")

	//assume our context has been given a deadline if needed
	c, _, err := websocket.DefaultDialer.DialContext(ctx, urlStr, nil)
	//	defer c.Close()

	if err != nil {
		log.WithField("error", err).Error("Dialing")
		return err
	}

	// assume we are conntected?
	r.Stats.ConnectedAt = time.Now()
	//close(r.connected) //signal that we've connected

	log.WithField("To", u).Info("Connected")

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
				log.WithField("info", err).Info("Reading")
				close(readClosed)
				break LOOP
			}
			// optionally forward messages
			if r.ForwardIncoming {
				r.In <- WsMessage{Data: data, Type: mt}
				//update stats
				r.Stats.Rx.Bytes.Add(float64(len(data)))
				r.Stats.Rx.Dt.Add(time.Since(r.Stats.Rx.Last).Seconds())
				r.Stats.Rx.Last = time.Now()
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
				log.WithField("error", err).Error("Writing")
				break LOOPWRITING
			}
			//update stats
			r.Stats.Tx.Bytes.Add(float64(len(msg.Data)))
			r.Stats.Tx.Dt.Add(time.Since(r.Stats.Tx.Last).Seconds())
			r.Stats.Tx.Last = time.Now()

		case <-ctx.Done(): // context has finished, either timeout or cancel
			//TODO - do we need to do this?
			// Cleanly close the connection by sending a close message
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.WithField("error", err).Error("Closing")
			} else {
				log.Info("Closed")
			}
			c.Close()
			break LOOPWRITING
		}
	}

	return err

}
