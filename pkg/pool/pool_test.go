package pool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func mockTime(now *int64) int64 {
	return *now
}

func TestNewPool(t *testing.T) {

	time := time.Now().Unix()

	p := NewPool("test").WithNow(func() int64 { return mockTime(&time) })

	assert.Equal(t, time, p.getTime())

	time = time + 3600

	assert.Equal(t, time, p.getTime())

}
