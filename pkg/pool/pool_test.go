package pool

import (
	"encoding/json"
	"fmt"
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

func TestAddRequestActivity(t *testing.T) {

	time := time.Now().Unix()
	starttime := time

	p := NewPool("test").WithNow(func() int64 { return mockTime(&time) })

	assert.Equal(t, time, p.getTime())

	old := NewActivity("act1", time-1)

	err := p.AddActivity(old)
	assert.Error(t, err)

	a := NewActivity("a", time+3600)
	assert.True(t, a.ID != "")
	assert.True(t, len(a.ID) >= 35)

	err = p.AddActivity(a)
	assert.NoError(t, err)

	b := NewActivity("b", time+7200)
	assert.True(t, b.ID != "")
	assert.True(t, len(b.ID) >= 35)
	assert.NotEqual(t, a.ID, b.ID)

	err = p.AddActivity(b)
	assert.NoError(t, err)

	ids := p.GetActivityIDs()

	founda := false
	foundb := false

	for _, id := range ids {
		if id == a.ID {
			founda = true
		}
		if id == b.ID {
			foundb = true
		}
	}

	assert.True(t, founda && foundb, "IDs did not match")

	if !(founda && foundb) {
		fmt.Println(ids)

		prettya, err := json.MarshalIndent(a, "", "\t")
		assert.NoError(t, err)
		prettyb, err := json.MarshalIndent(b, "", "\t")
		assert.NoError(t, err)

		fmt.Println(string(prettya))
		fmt.Println(string(prettyb))
	}

	aa, err := p.GetActivityByID(a.ID)

	assert.Equal(t, a.Name, aa.Name)

	assert.True(t, p.ActivityExists(a.ID))
	assert.True(t, p.ActivityExists(b.ID))
	assert.False(t, p.ActivityInUse(a.ID))
	assert.False(t, p.ActivityInUse(b.ID))

	at, err := p.ActivityNextAvailableTime(a.ID)
	assert.NoError(t, err)
	assert.Equal(t, time, at)

	bt, err := p.ActivityNextAvailableTime(b.ID)
	assert.NoError(t, err)
	assert.Equal(t, time, bt)

	wait, err := p.ActivityWaitAny()
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), wait)

	id, err := p.ActivityRequestAny(5000) //rules out a
	assert.NoError(t, err)
	assert.Equal(t, b.ID, id)

	assert.False(t, p.ActivityInUse(a.ID))
	assert.True(t, p.ActivityInUse(b.ID))

	at, err = p.ActivityNextAvailableTime(a.ID)
	assert.NoError(t, err)
	assert.Equal(t, time, at)

	bt, err = p.ActivityNextAvailableTime(b.ID)
	assert.NoError(t, err)
	assert.Equal(t, time+5000, bt)

	wait, err = p.ActivityWaitAny()
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), wait)

	wait, err = p.ActivityWaitDuration(5000) //none left, b will be expired before session ends
	assert.Error(t, err)

	wait, err = p.ActivityWaitDuration(1000) //a is left
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), wait)

	id, err = p.ActivityRequestAny(2000)
	assert.NoError(t, err)
	assert.Equal(t, a.ID, id)

	wait, err = p.ActivityWaitDuration(2000) //rules out a, because it only has 1600 left
	assert.NoError(t, err)
	assert.Equal(t, uint64(5000), wait)

	id, err = p.ActivityRequestAny(1000) // none left!
	assert.Error(t, err)

	time = starttime + 2001 // a just finished

	assert.False(t, p.ActivityInUse(a.ID))
	assert.True(t, p.ActivityInUse(b.ID))

	time = starttime + 3601 //a just expired

	assert.False(t, p.ActivityExists(a.ID))
	assert.True(t, p.ActivityExists(b.ID))

}
