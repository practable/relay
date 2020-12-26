package counter

import (
	"sync"
)

type Counter struct {
	mux   sync.Mutex
	count int
}

func (c *Counter) Increment() {
	c.mux.Lock()
	c.count++
	c.mux.Unlock()
}

func (c *Counter) Read() int {
	c.mux.Lock()
	count := c.count
	c.mux.Unlock()
	return count
}

func (c *Counter) Reset() {
	c.mux.Lock()
	c.count = 0
	c.mux.Unlock()
}

func New() *Counter {
	return &Counter{count: 0}
}
