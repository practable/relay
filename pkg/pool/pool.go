package pool

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

func NewPoolStore() *PoolStore {
	return &PoolStore{
		&sync.RWMutex{},
		make(map[string]*Group),
		make(map[string]*Pool),
		[]byte(""),
		3600,
		func() int64 { return time.Now().Unix() },
	}
}

func (p *PoolStore) WithSecret(secret string) *PoolStore {
	p.Lock()
	defer p.Unlock()
	p.Secret = []byte(secret)
	return p
}

func (p *PoolStore) WithNow(now func() int64) *PoolStore {
	p.Lock()
	defer p.Unlock()
	p.Now = now
	return p
}

func (p *PoolStore) WithBookingTokenDuration(duration int64) *PoolStore {
	p.Lock()
	defer p.Unlock()
	p.BookingTokenDuration = duration
	return p
}

// GetTime allows flexibility in choosing clock source
// default is internal clock of the server
func (p *PoolStore) GetTime() int64 {
	return p.Now()
}

func (p *PoolStore) GetSecret() []byte {
	p.RLock()
	defer p.RUnlock()
	return p.Secret
}

func (p *PoolStore) GetGroupByID(id string) (*Group, error) {
	p.RLock()
	defer p.RUnlock()

	group, ok := p.Groups[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return group, nil
}

func (p *PoolStore) GetGroupsByNamePrefix(prefix string) ([]*Group, error) {
	p.RLock()
	defer p.RUnlock()

	groups := []*Group{}

	for _, v := range p.Groups {
		if strings.HasPrefix(v.Name, prefix) {
			groups = append(groups, v)
		}
	}

	if len(groups) == 0 {
		return groups, errors.New("not found")
	}

	return groups, nil
}

func (p *PoolStore) GetGroupsByName(name string) ([]*Group, error) {
	p.RLock()
	defer p.RUnlock()

	groups := []*Group{}

	for _, v := range p.Groups {
		if v.Name == name {
			groups = append(groups, v)
		}
	}

	if len(groups) == 0 {
		return groups, errors.New("not found")
	}

	return groups, nil
}

func (p *PoolStore) GetPoolByID(id string) (*Pool, error) {
	p.RLock()
	defer p.RUnlock()

	pool, ok := p.Pools[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return pool, nil
}

func (p *PoolStore) GetPoolsByNamePrefix(prefix string) ([]*Pool, error) {
	p.RLock()
	defer p.RUnlock()

	pools := []*Pool{}

	for _, v := range p.Pools {
		if strings.HasPrefix(v.Name, prefix) {
			pools = append(pools, v)
		}
	}

	if len(pools) == 0 {
		return pools, errors.New("not found")
	}

	return pools, nil
}

func (p *PoolStore) GetPoolsByName(name string) ([]*Pool, error) {
	p.RLock()
	defer p.RUnlock()

	pools := []*Pool{}

	for _, v := range p.Pools {
		if v.Name == name {
			pools = append(pools, v)
		}
	}

	if len(pools) == 0 {
		return pools, errors.New("not found")
	}

	return pools, nil
}

func (p *PoolStore) DeletePool(pool *Pool) {
	p.Lock()
	defer p.Unlock()
	pools := p.Pools
	delete(pools, pool.ID)
	p.Pools = pools
}

func (p *PoolStore) DeleteGroup(group *Group) {
	p.Lock()
	defer p.Unlock()
	groups := p.Groups
	delete(groups, group.ID)
	p.Groups = groups
}

func (p *PoolStore) AddPool(pool *Pool) {
	p.Lock()
	defer p.Unlock()
	pools := p.Pools
	pools[pool.ID] = pool
	p.Pools = pools
}

func (p *PoolStore) AddGroup(group *Group) {
	p.Lock()
	defer p.Unlock()
	groups := p.Groups
	groups[group.ID] = group
	p.Groups = groups
}

func (p *PoolStore) SetSecret(secret string) {
	p.Lock()
	defer p.Unlock()
	p.Secret = []byte(secret)
}

func NewGroup(name string) *Group {

	group := &Group{
		&sync.RWMutex{},
		*NewDescription(name),
		[]*Pool{},
	}
	return group
}

func (g *Group) WithID(id string) *Group {
	g.Lock()
	defer g.Unlock()
	g.ID = id
	return g
}

func (g *Group) GetID() string {
	g.Lock()
	defer g.Unlock()
	return g.ID
}

func (g *Group) WithPool(pool *Pool) *Group {
	g.Lock()
	defer g.Unlock()
	p := g.Pools
	p = append(p, pool)
	g.Pools = p
	return g
}

func (g *Group) WithPools(pools []*Pool) *Group {
	g.Lock()
	defer g.Unlock()
	p := g.Pools
	p = append(p, pools...)
	g.Pools = p
	return g
}

func (g *Group) AddPool(pool *Pool) {
	g.Lock()
	defer g.Unlock()
	p := g.Pools
	p = append(p, pool)
	g.Pools = p
}

func (g *Group) DeletePool(pool *Pool) {
	g.Lock()
	defer g.Unlock()

	// groups are small so performance optimisation not required
	pools := []*Pool{}
	for _, p := range g.Pools {
		if p != pool {
			pools = append(pools, p)
		}
	}
	g.Pools = pools
}

func (g *Group) AddPools(pools []*Pool) {
	g.Lock()
	defer g.Unlock()
	p := g.Pools
	p = append(p, pools...)
	g.Pools = p
}
func (g *Group) GetPools() []*Pool {
	g.Lock()
	defer g.Unlock()
	return g.Pools
}

func NewPool(name string) *Pool {

	pool := &Pool{
		&sync.RWMutex{},
		*NewDescription(name),
		make(map[string]*Activity),
		make(map[string]int64),
		make(map[string]int64),
		func() int64 { return time.Now().Unix() },
	}

	return pool
}

// getTime is an internal test function to check on time mocking
func (p *Pool) getTime() int64 {
	return p.Now()
}

func (p *Pool) WithNow(now func() int64) *Pool {
	p.Lock()
	defer p.Unlock()
	p.Now = now
	return p 
}

func (p *Pool) WithID(id string) *Pool {
	p.Lock()
	defer p.Unlock()
	p.ID = id
	return p
}

func (p *Pool) GetID() string {
	p.Lock()
	defer p.Unlock()
	return p.ID
}

func NewDescription(name string) *Description {
	return &Description{
		Name: name,
		ID:   uuid.New().String(),
	}
}

func (d *Description) WithID(id string) *Description {
	d.ID = id
	return d
}

func NewActivity(name string, expires int64) *Activity {
	return &Activity{
		&sync.RWMutex{},
		*NewDescription(name),
		expires,
		make(map[string]*Stream),
		Permission{},
	}
}

func (a *Activity) WithID(id string) *Activity {
	a.Lock()
	defer a.Unlock()
	a.ID = id
	return a
}

func (a *Activity) WithPermission(p Permission) *Activity {
	a.Lock()
	defer a.Unlock()
	a.Permission = p
	return a
}

func (a *Activity) GetPermission() Permission {
	a.RLock()
	defer a.RUnlock()
	return a.Permission
}

func (a *Activity) SetID(id string) {
	a.Lock()
	defer a.Unlock()
	a.ID = id
}

func (a *Activity) GetID() string {
	a.Lock()
	defer a.Unlock()
	return a.ID
}

func (a *Activity) SetPermission(p Permission) {
	a.Lock()
	defer a.Unlock()
	a.Permission = p
}

func (a *Activity) AddID() string {
	a.Lock()
	defer a.Unlock()
	id := uuid.New().String()
	a.ID = id
	return id
}

func (a *Activity) AddStream(key string, stream *Stream) {
	a.Lock()
	defer a.Unlock()
	s := a.Streams
	s[key] = stream
	a.Streams = s
}

func (p *Pool) AddActivity(activity *Activity) error {

	p.RemoveStaleEntries()

	if activity == nil {
		return errors.New("nil pointer to activity")
	}

	if activity.ExpiresAt <= p.Now() {
		return errors.New("activity already expired")
	}

	p.Lock()
	defer p.Unlock()

	a := p.Activities
	a[activity.ID] = activity
	p.Activities = a

	v := p.Available
	v[activity.ID] = activity.ExpiresAt
	p.Available = v

	return nil

}

func (p *Pool) DeleteActivity(activity *Activity) {
	p.Lock()
	defer p.Unlock()
	act := p.Activities
	delete(act, activity.ID)
	p.Activities = act
	av := p.Available
	delete(av, activity.ID)
	p.Available = av
}

func (p *Pool) GetActivityIDs() []string {

	p.RemoveStaleEntries()

	p.RLock()
	defer p.RUnlock()

	ids := []string{}

	for k, _ := range p.Activities {
		ids = append(ids, k)
	}

	return ids

}

func (p *Pool) CountInUse() int {
	p.RemoveStaleEntries()
	p.RLock()
	defer p.RUnlock()
	return len(p.InUse)
}

func (p *Pool) CountAvailable() int {
	p.RemoveStaleEntries()
	p.RLock()
	defer p.RUnlock()
	return len(p.Available)
}

func (p *Pool) GetActivityByID(id string) (*Activity, error) {

	p.RemoveStaleEntries()

	p.RLock()
	defer p.RUnlock()
	a := p.Activities[id]
	if a == nil {
		return a, errors.New("not found")
	}
	return a, nil
}

func (p *Pool) ActivityExists(id string) bool {

	p.RemoveStaleEntries()

	p.RLock()
	defer p.RUnlock()

	_, ok := p.Activities[id]
	return ok
}

func (p *Pool) ActivityInUse(id string) bool {

	p.RemoveStaleEntries()

	p.RLock()
	defer p.RUnlock()

	_, ok := p.InUse[id]
	return ok
}

func (p *Pool) ActivityNextAvailableTime(id string) (int64, error) {

	p.RemoveStaleEntries()

	p.RLock()
	defer p.RUnlock()

	if _, ok := p.Activities[id]; !ok {
		return -1, errors.New("not found")
	}

	t, ok := p.InUse[id]

	if !ok {
		return p.Now(), nil
	}

	return t, nil

}

func (p *Pool) RemoveStaleEntries() {

	p.Lock()
	defer p.Unlock()

	now := p.Now()

	// remove stale InUse entries

	ids := []string{}

	for k, v := range p.InUse {
		if v <= now {
			ids = append(ids, k)
		}
	}
	inUse := p.InUse

	for _, id := range ids {
		delete(inUse, id)
	}

	p.InUse = inUse

	// remove stale Available entries

	ids = []string{}
	for k, v := range p.Available {
		if v <= now {
			ids = append(ids, k)
		}
	}

	available := p.Available
	activities := p.Activities

	for _, id := range ids {
		delete(available, id)
		delete(activities, id)
	}

	p.Available = available
	p.Activities = activities

}

func (p *Pool) ActivityWaitAny() (uint64, error) {
	return p.ActivityWaitDuration(1) // not much you can do in one sec but they did ask...
}

func (p *Pool) ActivityWaitDuration(duration uint64) (uint64, error) {

	now := p.Now()
	until := now + int64(duration)

	p.RemoveStaleEntries()

	p.RLock()
	defer p.RUnlock()

	// check if anything is free now
	var id string
	var waits []uint64

	for k, expires := range p.Available {
		if expires < until {
			continue // won't be available long enough to fulfill desired session length
		}
		if ready, ok := p.InUse[k]; ok {
			delay := ready - now
			if delay < 0 {
				continue // stale, should have been deleted
			}
			if (ready + int64(duration)) > expires {
				continue
			}
			waits = append(waits, uint64(delay))
			continue
		}
		id = k
		break
	}

	if id != "" {
		return 0, nil // there are free activities of the required duration
	}

	if len(waits) == 0 {
		return 0, errors.New("none available") //nothing will come free
	}

	// nothing free now, but check smallest waits

	minWait := waits[0]

	for _, wait := range waits {
		if wait < minWait {
			minWait = wait
		}
	}

	return minWait, nil

}

// GetAnActivityUntil returns the ID of a free activity and marks the activity
// as 'in use' until the time requested, or throws an error if no free activities.
func (p *Pool) ActivityRequestAny(duration uint64) (string, error) {

	until := p.Now() + int64(duration)

	p.RemoveStaleEntries()

	p.Lock()
	defer p.Unlock()

	// find a free activity, that is going to exist long enough

	var id string

	for k, v := range p.Available {
		if v < until {
			continue
		}
		if _, ok := p.InUse[k]; ok {
			continue
		}
		id = k
		break
	}

	if id == "" {
		return "", errors.New("none available")
	}

	i := p.InUse

	i[id] = until

	p.InUse = i

	return id, nil

}

// ActivityRequestUntilTime marks activity with given ID as being 'in use' until
// the time requested, but throws an error if the ID does not exist or is in-use already
func (p *Pool) ActivityRequest(duration uint64, id string) error {

	until := p.Now() + int64(duration)

	p.RemoveStaleEntries()

	p.Lock()
	defer p.Unlock()

	if _, ok := p.Activities[id]; !ok {
		return errors.New("not found")
	}

	if _, ok := p.InUse[id]; !ok {
		return errors.New("already in use")
	}

	i := p.InUse

	i[id] = until

	p.InUse = i

	return nil

}
