// Package crossbar provides a message broker with topics
package crossbar

import (
	"sync"
)

// Crossbar creates and runs a new crossbar instance
func Crossbar(config Config, closed <-chan struct{}, denied chan string, parentwg *sync.WaitGroup) {

	var wg sync.WaitGroup

	messagesToDistribute := make(chan message, 10) //TODO make buffer length configurable

	wg.Add(1)

	go handleConnections(closed, &wg, messagesToDistribute, denied, config)

	wg.Wait()

	parentwg.Done()

}
