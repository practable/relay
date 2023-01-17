package chanmap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAddDelete(t *testing.T) {

	s := New()

	var chnull chan struct{}
	ch0 := make(chan struct{})

	err := s.Add("", "c0", ch0)
	assert.Error(t, err)
	assert.Equal(t, "no parent", err.Error())

	err = s.Add("p0", "", ch0)
	assert.Error(t, err)
	assert.Equal(t, "no child", err.Error())

	err = s.Add("p0", "c0", chnull)
	assert.Error(t, err)
	assert.Equal(t, "no channel", err.Error())

	// add first child
	err = s.Add("p0", "c0", ch0)
	assert.NoError(t, err)

	// add second child
	ch1 := make(chan struct{})
	err = s.Add("p0", "c1", ch1)
	assert.NoError(t, err)

	// use coroutines to check if channels (not) closed within some reasonable time limit
	go func() {
		select {
		case <-time.After(10 * time.Millisecond):
			t.Error("channel ch0 not closed as expected")
		case <-ch0:
			//pass (channel closed)
		}
	}()

	go func() {
		select {
		case <-time.After(10 * time.Millisecond):
			// pass (channel was not closed yet)
		case <-ch0:
			t.Error("channel ch1 closed unexpectedly")
		}
	}()

	err = s.DeleteAndClose("c0")
	assert.NoError(t, err)
	err = s.Delete("c1")
	assert.NoError(t, err)
}
