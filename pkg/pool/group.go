package pool

import (
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/timdrysdale/relay/pkg/booking/models"
)

func NewGroupFromModel(mg *models.Group) (*Group, error) {

	if mg == nil {
		return nil, errors.New("group is empty")
	}

	if mg.Description == nil {
		return nil, errors.New("description is empty")
	}

	if mg.Description.Name == nil {
		return nil, errors.New("name is empty")
	}

	d := NewDescriptionFromModel(mg.Description)

	group := &Group{
		&sync.RWMutex{},
		*d,
		[]*Pool{},
	}
	return group, nil
}

func (g *Group) SetNewRandomID() *Group {
	g.Lock()
	defer g.Unlock()
	g.ID = uuid.New().String()
	return g
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

func (g *Group) SetPools(pools []*Pool) {
	g.Lock()
	defer g.Unlock()
	g.Pools = pools
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

func (g *Group) GetPoolIDs() []string {

	pools := g.GetPools()

	poolIDs := []string{}

	for _, p := range pools {
		poolIDs = append(poolIDs, p.ID)
	}

	return poolIDs

}
