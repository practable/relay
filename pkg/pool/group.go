package pool

import "sync"

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
