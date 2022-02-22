package pool

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"
)

// NewStore creates a new Store with default settings
func NewStore() *Store {
	return &Store{
		&sync.RWMutex{},
		make(map[string]*Group),
		make(map[string]*Pool),
		[]byte(""),
		3600,
		func() int64 { return time.Now().Unix() },
	}
}

// Reset removes all pools and groups but does not change min/max session durations
func (p *Store) Reset() {

	p.Groups = make(map[string]*Group)
	p.Pools = make(map[string]*Pool)
}

// PostImportEssential sets up mutexes and Now() functions
func (p *Store) PostImportEssential() {
	// we're creating mutexes for first time, so we
	// don't need to take the lock just now.
	// This should be done before making
	// this the "live" Store or else
	// other conurrent handlers will try to take
	// locks with pointers to mutexes that don't exist...
	// TODO ... consider stopping other handlers during import.
	// Could set a flag for validateHeader ...
	// risk locking ourselves out though... TBC ....

	// Store
	p.RWMutex = &sync.RWMutex{}
	p.Now = func() int64 { return time.Now().Unix() }

	// Groups
	for _, g := range p.Groups {
		g.RWMutex = &sync.RWMutex{}
	}

	// Pools -> Activities -> Streams
	for _, pool := range p.Pools {
		pool.RWMutex = &sync.RWMutex{}
		pool.Now = func() int64 { return time.Now().Unix() }

		for _, act := range pool.Activities {
			act.RWMutex = &sync.RWMutex{}

			for _, s := range act.Streams {
				s.RWMutex = &sync.RWMutex{}
			}
		}
	}
}

// CopyStore copys pools, groups, secret, booking token duration and now function from one Store to another
func CopyStore(from, to *Store) {
	to.RWMutex = from.RWMutex
	to.Groups = from.Groups
	to.Pools = from.Pools
	to.Secret = from.Secret
	to.BookingTokenDuration = from.BookingTokenDuration
	to.Now = from.Now
}

// ExportAll marshalls the Store to JSON and returns the []byte representation
// note that elements such as mutexes cannot be serialised and are omitted
func (p *Store) ExportAll() ([]byte, error) {
	return json.Marshal(p)
}

// ImportAll imports a Store from a serialised JSON representation in []byte format
// It creates all necessary mutexes so the store can operate properly after the import
func ImportAll(b []byte) (*Store, error) {

	new := &Store{}

	err := json.Unmarshal(b, new)

	if err != nil {
		return nil, err
	}

	new.PostImportEssential()

	return new, nil

}

// PostImportSetNow applies a custom Now() func to the poolstore
// and all pools - useful for mocking time in tests which
// involve import/export
func (p *Store) PostImportSetNow(now func() int64) {
	p.Lock()
	defer p.Unlock()
	p.Now = now
	for _, pool := range p.Pools {
		pool.Lock()
		pool.Now = now
		pool.Unlock()
	}
}

// WithSecret sets the secret used by the booking store
func (p *Store) WithSecret(secret string) *Store {
	p.Lock()
	defer p.Unlock()
	p.Secret = []byte(secret)
	return p
}

// WithNow sets the function that returns the datetime in seconds at the current time (useful for testing)
func (p *Store) WithNow(now func() int64) *Store {
	p.Lock()
	defer p.Unlock()
	p.Now = now
	return p
}

// WithBookingTokenDuration sets the duration for which booking tokens are valid
func (p *Store) WithBookingTokenDuration(duration int64) *Store {
	p.Lock()
	defer p.Unlock()
	p.BookingTokenDuration = duration
	return p
}

// GetTime allows flexibility in choosing clock source
// default is internal clock of the server
func (p *Store) GetTime() int64 {
	return p.Now()
}

// GetSecret returns the secret used to check tokens
func (p *Store) GetSecret() []byte {
	p.RLock()
	defer p.RUnlock()
	return p.Secret
}

//GetGroupByID returns a pointer to the Group with the given ID, or an error
func (p *Store) GetGroupByID(id string) (*Group, error) {
	p.RLock()
	defer p.RUnlock()

	group, ok := p.Groups[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return group, nil
}

// GetGroupsByNamePrefix returns groups with a given string at the start of their name, or an error
func (p *Store) GetGroupsByNamePrefix(prefix string) ([]*Group, error) {
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

// GetGroupsByName returns all groups with the given name
func (p *Store) GetGroupsByName(name string) ([]*Group, error) {
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

// GetPoolByID returns a pointer to a Pool with the given ID, or an error
func (p *Store) GetPoolByID(id string) (*Pool, error) {
	p.RLock()
	defer p.RUnlock()

	pool, ok := p.Pools[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return pool, nil
}

// GetPoolsByNamePrefix returns all pools with names starting with a given string
func (p *Store) GetPoolsByNamePrefix(prefix string) ([]*Pool, error) {
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

// GetPoolsByName returns all pools with a given name
func (p *Store) GetPoolsByName(name string) ([]*Pool, error) {
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

// GetAllPools returns all pools in the Store
func (p *Store) GetAllPools() []*Pool {
	p.RLock()
	defer p.RUnlock()

	pools := []*Pool{}

	for _, p := range p.Pools {
		pools = append(pools, p)
	}
	return pools
}

// GetAllPoolCount returns a count of how many pools are in the store
func (p *Store) GetAllPoolCount() int {
	p.RLock()
	defer p.RUnlock()
	return len(p.Pools)
}

// GetAllGroupsCount returns a count of how many groups there are in the store
func (p *Store) GetAllGroupsCount() int {
	p.RLock()
	defer p.RUnlock()
	return len(p.Groups)
}

// GetAllPoolIDs returns an array containing all the IDs of pools in the store
func (p *Store) GetAllPoolIDs() []string {
	p.RLock()
	defer p.RUnlock()

	ids := []string{}

	for k := range p.Pools {
		ids = append(ids, k)
	}
	return ids
}

// DeletePool removes a pool from the store
func (p *Store) DeletePool(pool *Pool) {
	p.Lock()
	defer p.Unlock()
	pools := p.Pools
	delete(pools, pool.ID)
	p.Pools = pools
}

// DeleteGroup removes a group from the store
func (p *Store) DeleteGroup(group *Group) {
	p.Lock()
	defer p.Unlock()
	groups := p.Groups
	delete(groups, group.ID)
	p.Groups = groups
}

// AddPool adds a pool to the Store  (but does not associate it with a group)
func (p *Store) AddPool(pool *Pool) {
	p.Lock()
	defer p.Unlock()
	pools := p.Pools
	pools[pool.ID] = pool
	p.Pools = pools
}

// AddGroup adds a group to the Store
func (p *Store) AddGroup(group *Group) {
	p.Lock()
	defer p.Unlock()
	groups := p.Groups
	groups[group.ID] = group
	p.Groups = groups
}

// SetSecret sets the secret used to check token validity
// and sign tokens for streaming
func (p *Store) SetSecret(secret string) {
	p.Lock()
	defer p.Unlock()
	p.Secret = []byte(secret)
}

// GetAvailableActivitiesCount returns a count of how many activities are available
func (p *Store) GetAvailableActivitiesCount() int {
	p.Lock()
	defer p.Unlock()

	avail := 0

	for _, pool := range p.Pools {
		avail = avail + len(pool.Activities)
	}

	return avail

}
