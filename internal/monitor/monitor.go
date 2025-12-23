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
	"math/big"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/websocket"
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
func Monitor(closed <-chan struct{}, parentwg *sync.WaitGroup, config Config) {
	log.Info("Starting relay monitor")

	// slightly awkward transition from wg to context cancellation style, but
	// this way we retain consistency with other commands
	// but we get a bit easier time testing, as we can cancel contexts directly
	// - we don't need to wait for the monitor to finish up gracefully before exiting

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-closed
		cancel()

	}()

	monitor(ctx, config)
	log.Info("Relay monitor stopped")

	parentwg.Done()

}

func monitor(ctx context.Context, config Config) {

	for {
		select {
		case <-ctx.Done():
			return
		default:
			subctx, cancel := context.WithCancel(ctx)

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
	token, err := NewToken(config, time.Now())

	if err != nil {
		return errors.New("error creating token")
	}

	session := config.RelayAudience + "/session/" + config.Topic

	// connect to relay from first client
	tx := reconws.New()
	rx := reconws.New()

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
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// proceed
				<-time.After(config.Interval)
				now := big.NewInt(time.Now().UnixNano()).Bytes()
				tx.Out <- (reconws.WsMessage{Data: []byte(now), Type: websocket.BinaryMessage})
				log.Trace("tx sent message")
			}

			time.Sleep(config.Interval)

		}

	}()

	// receive messages in a loop, and check and log the latency
	go func() {
		missCount := 0
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-rx.In:
				if !ok {
					log.Error("rx receive channel closed")
					return
				}
				if msg.Type != websocket.BinaryMessage {
					log.Warnf("rx received non-binary message of type %d", msg.Type)
					continue
				}
				b := big.NewInt(0).SetBytes(msg.Data)
				sentTime := time.Unix(0, b.Int64())

				if err != nil {
					log.Errorf("error parsing sent time: %s", err.Error())
					continue
				}
				latency := time.Since(sentTime)
				// don't want to pollute logs with too much info, so use debug level
				log.Debugf("message latency: %s", latency.String())
				// check latency against threshold
				if latency > config.LatencyThreshold {
					// increment miss count
					missCount++
					log.Warnf("latency %s exceeds threshold %s (miss count %d)", latency.String(), config.LatencyThreshold.String(), missCount)
					if missCount >= config.TriggerAfterMisses {
						// trigger command
						log.Infof("triggering command: %s", config.Command)
						// reset miss count
						missCount = 0
						// execute command
						go func() {
							err := executeCommand(config.Command)
							if err != nil {
								log.Errorf("error executing command: %s", err.Error())
							}
						}()
						// avoid restarting the relay immediately so it's not stuck forever in a start-up/restart loop
						<-time.After(config.NoRetriggerWithin)
					}
				} else {
					// reset miss count because latency was ok
					missCount = 0
				}
			}
		}
	}()

	<-ctx.Done()
	return nil

}

// could mock this for testing, but easier to get it to touch a file in the current dir
func executeCommand(cmd string) error {
	// variable expansion first
	// execute the command
	expanded := os.ExpandEnv(cmd)
	log.Info("executing command: " + expanded)
	args := strings.Fields(expanded)
	c := exec.Command(args[0], args[1:]...)
	return c.Run()

}

// func NewToken creates a new JWT token for the relay
// time is provided as a parameter to allow for easier testing
func NewToken(c Config, now time.Time) (string, error) {
	// create a token
	var claims permission.Token
	iat := now.Unix()
	nbf := now.Add(-1 * time.Second).Unix()
	exp := now.Add(c.ReconnectEvery).Unix()

	claims.IssuedAt = jwt.NewNumericDate(time.Unix(iat, 0))
	claims.NotBefore = jwt.NewNumericDate(time.Unix(nbf, 0))
	claims.ExpiresAt = jwt.NewNumericDate(time.Unix(exp, 0))
	claims.Audience = jwt.ClaimStrings{c.RelayAudience}
	claims.BookingID = "relay-monitor"
	claims.Topic = c.Topic
	claims.ConnectionType = "session"
	claims.Scopes = []string{"read", "write"}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(c.RelaySecret))

}
