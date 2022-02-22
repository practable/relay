package counter

import (
	"sync"
)

// Counter represents an integer count
type Counter struct {
	mux   sync.Mutex
	count int
}

// Increment increases count by one
func (c *Counter) Increment() {
	c.mux.Lock()
	c.count++
	c.mux.Unlock()
}

// Read returns current counter value
func (c *Counter) Read() int {
	c.mux.Lock()
	count := c.count
	c.mux.Unlock()
	return count
}

// Reset sets count to zero
func (c *Counter) Reset() {
	c.mux.Lock()
	c.count = 0
	c.mux.Unlock()
}

// New returns pointer to Counter struct
func New() *Counter {
	return &Counter{count: 0}
}
