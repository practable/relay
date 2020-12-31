package shellrelay

import (
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/relay/pkg/shellaccess"
	"github.com/timdrysdale/relay/pkg/shellbar"
	"github.com/timdrysdale/relay/pkg/ttlcode"
)

func Relay(closed <-chan struct{}, parentwg *sync.WaitGroup, accessPort, relayPort int, audience, secret, target string) {

	var wg sync.WaitGroup

	cs := ttlcode.NewDefaultCodeStore()

	config := shellbar.Config{
		Listen:    relayPort,
		Audience:  target,
		CodeStore: cs,
	}

	wg.Add(1)
	go shellbar.Shellbar(config, closed, &wg)

	wg.Add(1)
	go shellaccess.API(closed, &wg, accessPort, audience, secret, target, cs)

	wg.Wait()
	parentwg.Done()
	log.Trace("Relay done")
}
