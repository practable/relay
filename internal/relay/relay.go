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

// Config holds the relay server paramters
type Config struct {
	AccessPort       int
	AllowNoBookingID bool
	Audience         string
	BufferSize       int64
	PruneEvery       time.Duration
	RelayPort        int
	Secret           string
	StatsEvery       time.Duration
	Target           string
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

	if config.BufferSize < 1 || config.BufferSize > 512 {
		log.WithFields(log.Fields{"requested": config.BufferSize, "actual": 256}).Warn("Overriding configured buffer size because out of range 1-512")
		config.BufferSize = 256
	}

	if config.StatsEvery < time.Duration(time.Second) {
		log.WithFields(log.Fields{"requested": config.StatsEvery, "actual": "1s"}).Warn("Overriding configured stats every because smaller than 1s")
		config.StatsEvery = time.Duration(time.Second) //we have to balance fast testing vs high CPU load in production if too short
	}

	hub := crossbar.New()

	crossbarConfig := crossbar.Config{
		Listen:     config.RelayPort,
		Audience:   config.Target,
		BufferSize: config.BufferSize,
		CodeStore:  cs,
		DenyStore:  ds,
		Hub:        hub,
		StatsEvery: config.StatsEvery,
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
		Hub:              hub,
		Port:             config.AccessPort,
		Secret:           config.Secret,
		Target:           config.Target,
	}

	go access.API(closed, &wg, accessConfig) //accessPort, audience, secret, target, cs, allowNoBookingID)

	wg.Wait()
	parentwg.Done()
	log.Trace("Relay done")
}
