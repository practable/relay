package pool

import (
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/practable/relay/internal/booking/models"
)

// NewGroupFromModel returns a pointer to Group created from the API's model
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

// SetNewRandomID sets a random UUID as the Group ID
func (g *Group) SetNewRandomID() *Group {
	g.Lock()
	defer g.Unlock()
	g.ID = uuid.New().String()
	return g
}

// NewGroup returns a pointer to a new Group with a name, and randomly generated ID
func NewGroup(name string) *Group {

	group := &Group{
		&sync.RWMutex{},
		*NewDescription(name),
		[]*Pool{},
	}
	return group
}

// WithID sets the ID of the Group
func (g *Group) WithID(id string) *Group {
	g.Lock()
	defer g.Unlock()
	g.ID = id
	return g
}

// GetID returns the ID of the Group
func (g *Group) GetID() string {
	g.Lock()
	defer g.Unlock()
	return g.ID
}

// WithPool adds a pool to a Group
func (g *Group) WithPool(pool *Pool) *Group {
	g.Lock()
	defer g.Unlock()
	p := g.Pools
	p = append(p, pool)
	g.Pools = p
	return g
}

// SetPools replaces all pools in the group with the given array of pools
func (g *Group) SetPools(pools []*Pool) {
	g.Lock()
	defer g.Unlock()
	g.Pools = pools
}

// WithPools replaces all pools in the Group with the given array of pools
func (g *Group) WithPools(pools []*Pool) *Group {
	g.Lock()
	defer g.Unlock()
	p := g.Pools
	p = append(p, pools...)
	g.Pools = p
	return g
}

// AddPool adds the pool to the existing list of pools, all existing pools are retained
func (g *Group) AddPool(pool *Pool) {
	g.Lock()
	defer g.Unlock()
	p := g.Pools
	p = append(p, pool)
	g.Pools = p
}

// DeletePool removes a single pool from the Group, if it was present
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

// AddPools adds more than one pool to the Group, retaining existing pools
func (g *Group) AddPools(pools []*Pool) {
	g.Lock()
	defer g.Unlock()
	p := g.Pools
	p = append(p, pools...)
	g.Pools = p
}

// GetPools returns the current list of pools in the Group
func (g *Group) GetPools() []*Pool {
	g.Lock()
	defer g.Unlock()
	return g.Pools
}

// GetPoolIDs returns an array of all Pool IDs  in the Group
func (g *Group) GetPoolIDs() []string {

	pools := g.GetPools()

	poolIDs := []string{}

	for _, p := range pools {
		poolIDs = append(poolIDs, p.ID)
	}

	return poolIDs

}
