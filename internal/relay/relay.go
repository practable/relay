package relay

import (
	"sync"
	"time"

	"github.com/practable/relay/internal/access"
	"github.com/practable/relay/internal/crossbar"
	"github.com/practable/relay/internal/deny"
	"github.com/practable/relay/internal/ttlcode"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	AccessPort       int
	RelayPort        int
	Audience         string
	Secret           string
	Target           string
	AllowNoBookingID bool
	PruneEvery       time.Duration
}

// Relay runs a websocket relay
func Relay(closed <-chan struct{}, parentwg *sync.WaitGroup, config Config) {

	//(closed <-chan struct{}, parentwg *sync.WaitGroup, accessPort, relayPort int, audience, secret, target string, allowNoBookingID bool) {

	var wg sync.WaitGroup

	denied := make(chan string, 64)

	cs := ttlcode.NewDefaultCodeStore()
	ds := deny.New()

	go func() {
		for {
			select {
			case <-closed:
				break
			case <-time.After(config.PruneEvery):
				{
					ds.Prune()
				}
			}
		}
	}()

	crossbarConfig := crossbar.Config{
		Listen:    config.RelayPort,
		Audience:  config.Target,
		CodeStore: cs,
		DenyStore: ds,
	}

	wg.Add(1)
	go crossbar.Crossbar(crossbarConfig, closed, denied, &wg)

	wg.Add(1)

	accessConfig := access.Config{
		AllowNoBookingID: config.AllowNoBookingID,
		CodeStore:        cs,
		DenyStore:        ds,
		DenyChannel:      denied,
		Host:             config.Audience,
		Port:             config.AccessPort,
		Secret:           config.Secret,
		Target:           config.Target,
	}

	go access.API(closed, &wg, accessConfig) //accessPort, audience, secret, target, cs, allowNoBookingID)

	wg.Wait()
	parentwg.Done()
	log.Trace("Relay done")
}
