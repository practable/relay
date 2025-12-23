package monitor

import (
	"bufio"
	"bytes"
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

func TestRelay(t *testing.T) {

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

	// Start tests

	fmt.Printf("Starting relay monitor test\n")

	// teardown relay

	close(closed)
	wg.Wait()

}
