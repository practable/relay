package shellbar

import (
	"sync"
)

// Shellbar runs ssh relay with the given configuration
func Shellbar(config Config, closed <-chan struct{}, parentwg *sync.WaitGroup) {

	var wg sync.WaitGroup

	messagesToDistribute := make(chan message, 10) //TODO make buffer length configurable

	var topics topicDirectory

	topics.directory = make(map[string][]clientDetails)

	clientActionsChan := make(chan clientAction)

	wg.Add(2)

	go handleConnections(closed, &wg, clientActionsChan, messagesToDistribute, config)

	go handleClients(closed, &wg, &topics, clientActionsChan)

	wg.Wait()

	parentwg.Done()

}
