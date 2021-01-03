package pool

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

func NewPoolStore() *PoolStore {
	return &PoolStore{
		&sync.RWMutex{},
		make(map[string]*Pool),
	}
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
	}
}

func (a *Activity) WithID(id string) *Activity {
	a.Lock()
	defer a.Unlock()
	a.ID = id
	return a
}

func (a *Activity) SetID(id string) {
	a.Lock()
	defer a.Unlock()
	a.ID = id
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

func (p *Pool) GetActivityIDs() []string {
	p.RLock()
	defer p.RUnlock()

	ids := []string{}

	for k, _ := range p.Activities {
		ids = append(ids, k)
	}

	return ids

}

func (p *Pool) GetActivityByID(id string) (*Activity, error) {
	p.RLock()
	defer p.RUnlock()
	a := p.Activities[id]
	if a == nil {
		return a, errors.New("not found")
	}
	return a, nil
}

func (p *Pool) ActivityExists(id string) bool {

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
		return 0, nil
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

	for k, v := range p.Available {
		if v < until {
			continue // won't be available long enough to fulfill desired session length
		}
		if v, ok := p.InUse[k]; ok {
			delay := v - now
			if delay < 0 {
				continue // stale, should have been deleted
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
