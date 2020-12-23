package relay

import (
	"sync"

	"github.com/timdrysdale/relay/pkg/access"
	"github.com/timdrysdale/relay/pkg/crossbar"
)

func Relay(config Config, closed chan struct{}, parentwg *sync.WaitGroup) {

	var wg sync.WaitGroup

	messagesToDistribute := make(chan message, 10) //TODO make buffer length configurable

	var topics crossbar.TopicDirectory

	topics.directory = make(map[string][]clientDetails)

	clientActionsChan := make(chan clientAction)

	wg.Add(3)

	go crossbar.HandleConnections(closed, &wg, clientActionsChan, messagesToDistribute, config)

	go crossbar.HandleClients(closed, &wg, &topics, clientActionsChan)

	go access.API(closed, &wg, config.ApiPort, config.ApiHost, config.ApiSecret, *access.DefaultOptions())

	wg.Wait()

	parentwg.Done()

}
