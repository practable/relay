package pool

import (
	"errors"
	"strings"
	"sync"
	"time"
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

func (p *PoolStore) GetAllPools() []*Pool {
	p.RLock()
	defer p.RUnlock()

	pools := []*Pool{}

	for _, p := range p.Pools {
		pools = append(pools, p)
	}
	return pools
}

func (p *PoolStore) GetAllPoolIDs() []string {
	p.RLock()
	defer p.RUnlock()

	ids := []string{}

	for k, _ := range p.Pools {
		ids = append(ids, k)
	}
	return ids
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
