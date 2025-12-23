package monitor

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/phayes/freeport"
	"github.com/practable/relay/internal/relay"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestWithRelay(t *testing.T) {

	// Setup logging
	debug := false

	if debug {
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true, DisableColors: true})
		defer log.SetOutput(os.Stdout)

	} else {
		var ignore bytes.Buffer
		logignore := bufio.NewWriter(&ignore)
		log.SetOutput(logignore)
	}

	// Setup relay on local (free) port
	closed := make(chan struct{})
	var wg sync.WaitGroup

	ports, err := freeport.GetFreePorts(2)
	assert.NoError(t, err)

	relayPort := ports[0]
	accessPort := ports[1]

	audience := "http://[::]:" + strconv.Itoa(accessPort)
	target := "ws://127.0.0.1:" + strconv.Itoa(relayPort)

	fmt.Printf("audience:%s\n", audience)
	fmt.Printf("target:%s\n", target)

	secret := "testsecret"

	wg.Add(1)

	config := relay.Config{
		AccessPort:       accessPort,
		RelayPort:        relayPort,
		Audience:         audience,
		Secret:           secret,
		Target:           target,
		AllowNoBookingID: true,
		PruneEvery:       time.Duration(time.Minute),
	}

	go relay.Relay(closed, &wg, config) //accessPort, relayPort, audience, secret, target)

	time.Sleep(time.Second) // big safety margin to get crossbar running

	// clean up any prior test file
	_ = os.Remove("triggered")

	// Start tests
	fmt.Printf("Starting relay monitor test\n")
	ctx, cancel := context.WithCancel(context.Background())

	// these values would be too much load for production, but make testing quicker
	monitorConfig := Config{
		RelayAudience:      audience,
		RelaySecret:        secret,
		Command:            "touch triggered",
		Interval:           100 * time.Millisecond,
		LatencyThreshold:   10 * time.Millisecond,
		NoRetriggerWithin:  10 * time.Second, // we won't retrigger during this test.
		ReconnectEvery:     15 * time.Second, // we won't reconnect during this test.
		Topic:              "test-topic",
		TriggerAfterMisses: 1,
	}

	go monitor(ctx, monitorConfig) //run the monitor

	time.Sleep(500 * time.Millisecond) // let the monitor run a bit

	_, err = os.Stat("triggered")

	assert.Error(t, err, "expected file to not be there yet")

	cancel() // stop monitor

	monitorConfig.LatencyThreshold = time.Microsecond // set unrealistic threshold to force trigger

	ctx, cancel = context.WithCancel(context.Background())

	go monitor(ctx, monitorConfig) //run the monitor

	time.Sleep(500 * time.Millisecond) // let the monitor run a bit

	_, err = os.Stat("triggered")

	assert.NoError(t, err, "expected file to be there after trigger")

	time.Sleep(2 * time.Second) // collect some more logs
	// teardown test
	_ = os.Remove("triggered")
	cancel()      // stop monitor
	close(closed) //stop relay
	wg.Wait()

}
func TestExecuteCommand(t *testing.T) {
	debug := false

	if debug {
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true, DisableColors: true})
		defer log.SetOutput(os.Stdout)

	} else {
		var ignore bytes.Buffer
		logignore := bufio.NewWriter(&ignore)
		log.SetOutput(logignore)
	}

	err := executeCommand("touch triggered")
	assert.NoError(t, err)

	err = os.Remove("triggered")
	assert.NoError(t, err) // if the file isn't there, then we'll get an error

}

func TestToken(t *testing.T) {
	c := Config{
		RelayAudience: "https://app.practable.io/ed0/access",
		RelaySecret:   "xxx000",
		Topic:         "canary-st-data",
	}
	token, err := NewToken(c, time.Unix(1658306108, 0))

	assert.NoError(t, err)
	expected := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJib29raW5nX2lkIjoicmVsYXktbW9uaXRvciIsInRvcGljIjoiY2FuYXJ5LXN0LWRhdGEiLCJwcmVmaXgiOiJzZXNzaW9uIiwic2NvcGVzIjpbInJlYWQiLCJ3cml0ZSJdLCJhdWQiOlsiaHR0cHM6Ly9hcHAucHJhY3RhYmxlLmlvL2VkMC9hY2Nlc3MiXSwiZXhwIjoxNjU4MzA2MTA4LCJuYmYiOjE2NTgzMDYxMDcsImlhdCI6MTY1ODMwNjEwOH0.ScP6fv_ddXs0iQ2WY6oMGafgTKv7bZsp_0sVbHe7CNY"

	/*
		{
		  "alg": "HS256",
		  "typ": "JWT"
		}
		{
		  "booking_id": "relay-monitor",
		  "topic": "canary-st-data",
		  "prefix": "session",
		  "scopes": [
		    "read",
		    "write"
		  ],
		  "aud": [
		    "https://app.practable.io/ed0/access"
		  ],
		  "exp": 1658306108,
		  "nbf": 1658306107,
		  "iat": 1658306108
		}
	*/

	assert.NoError(t, err)
	assert.Equal(t, expected, token)

}
