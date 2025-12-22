/*
monitor is a websocket client that checks the latency of messages
it sends through a relay, triggering a command if the latency exceeds
a threshold

Copyright (C) 2025 Timothy Drysdale <timothy.d.drysdale@gmail.com>

*/

package monitor

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/practable/relay/internal/permission"
	"github.com/practable/relay/internal/reconws"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	Command            string
	Interval           time.Duration
	LatencyThreshold   time.Duration
	NoRetriggerWithin  time.Duration
	ReconnectEvery     time.Duration
	RelayAudience      string
	RelaySecret        string
	Topic              string
	TriggerAfterMisses int
}

// Monitor runs a websocket relay monitor
func Monitor(closed <-chan struct{}, parentwg *sync.WaitGroup, config Congfig) {
	log.Info("Starting relay monitor")

	// slightly awkward transition from wg to context cancellation style, but
	// this way we retain consistency with other commands
	// but we get a bit easier time testing, as we can cancel contexts directly
	// - we don't need to wait for the monitor to finish up gracefully before exiting

	ctx, cancel = context.WithCancel(context.Background())

	go func() {
		<-closed
		cancel()

	}()

	monitor(ctx, config)

	log.Info("Relay monitor stopped")
	parentwg.Done()

}

func monitor(ctx context.Context, config Config) error {

	for {
		select {
		case <-ctx.Done():
			return
		default:
			subctx, cancel = context.WithCancel(ctx)

			go func() {
				<-time.After(config.ReconnectEvery)
				cancel()
			}()

			err := runOnce(subctx, config)
			if err != nil {
				log.Errorf("error running monitor iteration: %s", err.Error())
				<-time.After(30 * time.Second) //wait before retrying
			}
		}
	}
}

func runOnce(ctx context.Context, config Config) error {

	// create a token
	token, err := NewToken(config)

	if err != nil {
		return errors.New("error creating token")
	}

	session := config.RelayAudience + "/session/" + config.Topic

	// connect to relay from first client
	tx := reconws.New()
	rx = reconws.New()

	// both connect to the same session
	go tx.ReconnectAuth(ctx, session, token)
	go rx.ReconnectAuth(ctx, session, token)

	select {
	case <-tx.Connected: //wait for connection to be made
		log.Debug("tx connected to relay")
	case <-time.After(time.Minute):
		return errors.New("timeout connecting tx to relay")
	}

	select {
	case <-rx.Connected: //wait for connection to be made
		log.Debug("rx connected to relay")
	case <-time.After(time.Minute):
		return errors.New("timeout connecting to relay")
	}

	// send messages on a loop
	
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
				// proceed
				<-time.After(config.Interval)
				now := time.Now().UnixNano().String()
				tx.Send(reconws.WsMessage{Data: []byte(now)})
			}

			time.Sleep(config.Interval)

		}
	}()

	// check and log the latency

	// if the latency exceeds threshold too many times,  trigger the command

}

func NewToken(c Config) (string, error) {
	// create a token
	var claims permission.Token
	now := time.Now().Unix()
	iat := now
	nbf := now.Add(-1 * time.Second).Unix()
	exp := now.Add(Config.ReconnectEvery).Unix()

	claims.IssuedAt = jwt.NewNumericDate(time.Unix(iat, 0))
	claims.NotBefore = jwt.NewNumericDate(time.Unix(nbf, 0))
	claims.ExpiresAt = jwt.NewNumericDate(time.Unix(exp, 0))
	claims.Audience = jwt.ClaimStrings{audience}
	claims.BookingID = "relay-monitor"
	claims.Topic = topic
	claims.ConnectionType = "session"
	claims.Scopes = []string{"read", "write"}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))

}
