package relay

import (
	"sync"

	"github.com/practable/relay/internal/access"
	"github.com/practable/relay/internal/crossbar"
	"github.com/practable/relay/internal/ttlcode"
	log "github.com/sirupsen/logrus"
)

// Relay runs a websocket relay
func Relay(closed <-chan struct{}, parentwg *sync.WaitGroup, accessPort, relayPort int, audience, secret, target string, allowNoBookingID bool) {

	var wg sync.WaitGroup

	cs := ttlcode.NewDefaultCodeStore()

	config := crossbar.Config{
		Listen:    relayPort,
		Audience:  target,
		CodeStore: cs,
	}

	wg.Add(1)
	go crossbar.Crossbar(config, closed, &wg)

	wg.Add(1)
	go access.API(closed, &wg, accessPort, audience, secret, target, cs, allowNoBookingID)

	wg.Wait()
	parentwg.Done()
	log.Trace("Relay done")
}
