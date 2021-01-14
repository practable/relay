// Package manifest provides an alternative data
// format for representing poolstores that is
// easier to edit by hand, because default options
// can be specified for activities within a given
// pool, where usually the details are similar
// except for the stream details
package manifest

import (
	"io"
	"io/ioutil"

	"github.com/dgrijalva/jwt-go"
	"github.com/timdrysdale/relay/pkg/permission"
	"github.com/timdrysdale/relay/pkg/pool"
	"gopkg.in/yaml.v2"
)

// Store represents all the pools and groups in the poolstore
type Manifest struct {

	// Groups represents all the groups in the poolstore
	Groups []*Group `yaml:"groups"`

	// Pools represents all the pools in the poolstore
	Pools []*Pool
}

type Group struct {

	// ID represents the PoolStore-provided ID, obtained on adding the group
	ID string

	Description *pool.Description

	// Pools represents all the pools in the group
	Pools []string
}

type Pool struct {

	// ID represents the PoolStore-provided ID, obtained on adding the pool
	ID string

	// Pool represents the pool
	Pool *pool.Pool

	// DefaultActivity represents the default properties of activities
	DefaultActivity *pool.Activity
}

func (m *Manifest) GetPoolStore() *pool.PoolStore {

	ps := pool.NewPoolStore()

	//TODO implement!

	return ps

}

func GetManifest(ps *pool.PoolStore) *Manifest {

	m := &Manifest{}

	// TODO - make sensible defaults  map the arguments, and treat as defaults those that only have one entry?

	//pmap := make(map[*pool.Pool]*Pool)

	for _, pp := range ps.Pools {
		mp := &Pool{
			ID:              pp.ID,
			Pool:            pp,
			DefaultActivity: pool.NewActivity("default", 0),
		}
		//pmap[pp] = mp

		m.Pools = append(m.Pools, mp)
	}

	for _, pg := range ps.Groups {

		mg := &Group{
			Description: &pg.Description,
			Pools:       []string{},
		}

		for _, pp := range pg.Pools {

			mg.Pools = append(mg.Pools, pp.ID)

			//if mp, ok := pmap[pp]; ok {
			//	mg.Pools = append(mg.Pools, mp)
			//}

		}

		m.Groups = append(m.Groups, mg)
	}

	return m

}

func LoadManifest(r io.Reader) (*pool.PoolStore, error) {

	buf, err := ioutil.ReadAll(r)

	if err != nil {
		return nil, err
	}

	m := &Manifest{}

	err = yaml.Unmarshal(buf, &m)

	ps := m.GetPoolStore()

	return ps, nil
}

func (m *Manifest) Write(w io.Writer) (int, error) {

	buf, err := yaml.Marshal(m)

	if err != nil {
		return 0, err
	}

	return w.Write(buf)
}

// Example provides a populated poolstore which can
// be used to create a template for users
// at runtime, as well as help with testing
func Example() *pool.PoolStore {

	duration := int64(2628000) //3 months

	ps := pool.NewPoolStore()

	name := "stuff"

	g0 := pool.NewGroup(name)

	ps.AddGroup(g0)

	p0 := pool.NewPool("stuff0")

	p0.DisplayInfo = pool.DisplayInfo{
		Short:   "The Good Stuff - Pool 0",
		Long:    "This stuff has some good stuff in it",
		Further: "https://example.com/further.html",
		Thumb:   "https://example.com/thumb.png",
		Image:   "https://example.com/img.png",
	}

	g0.AddPool(p0)
	ps.AddPool(p0)

	a := pool.NewActivity("a", ps.Now()+duration)

	p0.AddActivity(a)

	pt0 := permission.Token{
		ConnectionType: "session",
		Topic:          "123",
		Scopes:         []string{"read", "write"},
		StandardClaims: jwt.StandardClaims{
			Audience: "https://example.com",
		},
	}
	s0 := pool.NewStream("https://example.com/session/123data")
	s0.SetPermission(pt0)
	a.AddStream("data", s0)

	pt1 := permission.Token{
		ConnectionType: "session",
		Topic:          "456",
		Scopes:         []string{"read"},
		StandardClaims: jwt.StandardClaims{
			Audience: "https://example.com",
		},
	}
	s1 := pool.NewStream("https://example.com/session/456video")
	s1.SetPermission(pt1)
	a.AddStream("video", s1)

	du0 := pool.Description{
		DisplayInfo: pool.DisplayInfo{
			Short:   "The UI that's green",
			Long:    "This has some green stuff in it",
			Further: "https://example.com/further0.html",
			Thumb:   "https://example.com/thumb0.png",
			Image:   "https://example.com/img0.png",
		},
	}

	u0 := pool.NewUI("https://static.example.com/example.html?data={{data}}&video={{video}}").
		WithStreamsRequired([]string{"data", "video"}).
		WithDescription(du0)

	a.AddUI(u0)

	du1 := pool.Description{
		DisplayInfo: pool.DisplayInfo{
			Short:   "The UI that's blue",
			Long:    "This has some blue stuff in it",
			Further: "https://example.com/further1.html",
			Thumb:   "https://example.com/thumb1.png",
			Image:   "https://example.com/img1.png",
		},
	}

	u1 := pool.NewUI("https://static.example.com/other.html?data={{data}}&video={{video}}").
		WithStreamsRequired([]string{"data", "video"}).
		WithDescription(du1)

	a.AddUI(u1)

	return ps

}
