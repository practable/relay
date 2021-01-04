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

func TestAddRequestCountActivity(t *testing.T) {

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

	assert.Equal(t, 2, p.CountAvailable())
	assert.Equal(t, 0, p.CountInUse())

	id, err := p.ActivityRequestAny(5000) //rules out a
	assert.NoError(t, err)
	assert.Equal(t, b.ID, id)

	assert.False(t, p.ActivityInUse(a.ID))
	assert.True(t, p.ActivityInUse(b.ID))

	assert.Equal(t, 2, p.CountAvailable())
	assert.Equal(t, 1, p.CountInUse())

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

	assert.Equal(t, 1, p.CountAvailable())
	p.DeleteActivity(b)
	assert.Equal(t, 0, p.CountAvailable())

}

func TestAddGetDeletePools(t *testing.T) {

	ps := NewPoolStore().WithSecret("foo")

	assert.Equal(t, []byte("foo"), ps.GetSecret())

	p0 := NewPool("stuff0")
	p1 := NewPool("stuff1")
	p2 := NewPool("things")

	ps.AddPool(p0)
	ps.AddPool(p1)
	ps.AddPool(p2)

	pool, err := ps.GetPoolByID("definitelyNotAPoolIDBecauseNotAUUID")

	assert.Error(t, err)

	assert.Equal(t, "not found", err.Error())

	pool, err = ps.GetPoolByID(p0.ID)

	assert.NoError(t, err)

	assert.Equal(t, p0.Name, pool.Name)

	pools, err := ps.GetPoolsByName("stuff1")

	assert.Equal(t, 1, len(pools))

	assert.Equal(t, p1.Name, (pools[0]).Name)

	pools, err = ps.GetPoolsByNamePrefix("stuff")

	assert.Equal(t, 2, len(pools))

	ps.DeletePool(p0)
	pools, err = ps.GetPoolsByNamePrefix("stuff")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pools))
	ps.DeletePool(p1)
	pools, err = ps.GetPoolsByNamePrefix("stuff")
	assert.Error(t, err)
	assert.Equal(t, "not found", err.Error())

}

func TestAddGetDeleteGroups(t *testing.T) {

	ps := NewPoolStore().WithSecret("bar")

	assert.Equal(t, []byte("bar"), ps.GetSecret())

	g0 := NewGroup("stuff0")
	g1 := NewGroup("stuff1")
	g2 := NewGroup("things")

	ps.AddGroup(g0)
	ps.AddGroup(g1)
	ps.AddGroup(g2)

	group, err := ps.GetGroupByID("definitelyNotAGroupIDBecauseNotAUUID")

	assert.Error(t, err)

	assert.Equal(t, "not found", err.Error())

	group, err = ps.GetGroupByID(g0.ID)

	assert.NoError(t, err)

	assert.Equal(t, g0.Name, group.Name)

	groups, err := ps.GetGroupsByName("stuff1")

	assert.Equal(t, 1, len(groups))

	assert.Equal(t, g1.Name, (groups[0]).Name)

	groups, err = ps.GetGroupsByNamePrefix("stuff")

	assert.Equal(t, 2, len(groups))

	ps.DeleteGroup(g0)

	groups, err = ps.GetGroupsByNamePrefix("stuff")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(groups))

	ps.DeleteGroup(g1)

	groups, err = ps.GetGroupsByNamePrefix("stuff")
	assert.Error(t, err)
	assert.Equal(t, "not found", err.Error())

}

func TestAddGetPoolInGroup(t *testing.T) {

	g0 := NewGroup("stuff")
	g1 := NewGroup("things")

	p0 := NewPool("stuff0")
	p1 := NewPool("stuff1")
	p2 := NewPool("things0")
	p3 := NewPool("things1")

	g0.AddPool(p0)
	g0.AddPool(p1)
	g1.AddPools([]*Pool{p2, p3})

	pools0 := g0.GetPools()
	assert.Equal(t, 2, len(pools0))

	pools1 := g1.GetPools()
	assert.Equal(t, 2, len(pools1))

	g1.DeletePool(p2)
	pools1 = g1.GetPools()
	assert.Equal(t, 1, len(pools1))

	// delete deleted item causes no change
	g1.DeletePool(p2)
	pools1 = g1.GetPools()
	assert.Equal(t, 1, len(pools1))

	g1.DeletePool(p3)
	pools1 = g1.GetPools()
	assert.Equal(t, 0, len(pools1))

}

func TestAddPermissionsToActivity(t *testing.T) {

	p := Permission{
		Audience:       "https://relay-access.example.io",
		ConnectionType: "session",
		Topic:          "123",
	}

	a := NewActivity("pend00", time.Now().Unix()+1000).WithPermission(p)

	assert.Equal(t, p, a.GetPermission())

}
