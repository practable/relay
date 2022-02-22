package manifest

/*
	func (m *Manifest) GetPoolStore() *pool.PoolStore {

	ps := pool.NewPoolStore().
		WithBookingTokenDuration(m.BookingTokenDuration).
		WithSecret(m.Secret)

	// inflate and add pools, and correlate id with ptr
	// for construcing group later

	pmap := make(map[string]*pool.Pool)

	for _, mp := range m.Pools {
		p := mp.ToPool()
		pmap[mp.ID] = p
		ps.AddPool(p)
	}

	//TODO implement groups

	return ps

}

func (mp *Pool) ToPool() *pool.Pool {

	p := &pool.Pool{
		Description: mp.Description,
		MinSession:  mp.MaxSession,
		MaxSession:  mp.MaxSession,
	}
	if len(mp.Activities) < 1 {
		return p
	}

	p.Activities = make(map[string]*pool.Activity)

	for _, a := range mp.Activities {

		pa := &pool.Activity{
			Description: mp.ActivityDescription,
			ExpiresAt:   a.ExpiresAt,
			UI:          mp.UI,
			Streams:     make(map[string]*pool.Stream),
		}

		for k, v := range mp.Streams {

			s := v

			if _, ok := a.Tokens[k]; !ok {
				continue
			}

			s.Permission = a.Tokens[k]

			pa.Streams[k] = &s
		}

		p.Activities[pa.ID] = pa

	}

	return p

}

func PoolToManifestPool(p pool.Pool) Pool {

	mp := Pool{
		Description: p.Description,
		MinSession:  p.MaxSession,
		MaxSession:  p.MaxSession,
	}

	if len(p.Activities) < 1 {
		return mp
	}

	// get the first key in the map
	var key string
	for k, _ := range p.Activities {
		key = k
		break
	}

	// use that key/activity to populate the default activity
	a := p.Activities[key]

	mp.ActivityDescription = a.Description

	s := make(map[string]pool.Stream)

	for _, v := range a.Streams {
		v.Permission = permission.Token{} //ignore tokens in default representation
		s[v.For] = *v
	}

	mp.Streams = s

	//u := []*pool.UI{}

	//for _, v := range a.UI {
	//	u = append(u, *v)
	//}

	mp.UI = a.UI

	mas := []Activity{}

	for _, pa := range p.Activities {

		ma := Activity{
			ExpiresAt: pa.ExpiresAt,
			Tokens:    make(map[string]permission.Token),
		}

		for _, v := range pa.Streams {

			ma.Tokens[v.For] = v.Permission

		}

		mas = append(mas, ma)
	}

	mp.Activities = mas

	return mp

}

func GetManifest(ps *pool.PoolStore) *Manifest {

	m := &Manifest{
		Secret:               string(ps.Secret),
		BookingTokenDuration: ps.BookingTokenDuration,
	}

	// Manifest assumes that all activities in a pool are
	// identical EXCEPT for the permission token, so as
	// to keep the manifest as compact and efficient as
	// possible, and help the human editor!
	// It won't be true once multiple suppliers exist,
	// so we need to develop better manifest support
	// for when that happens.

	pmap := make(map[*pool.Pool]string)

	for _, pp := range ps.Pools {

		mp := PoolToManifestPool(*pp)
		pid := uuid.New().String()
		mp.ID = pid
		pmap[pp] = pid
		m.Pools = append(m.Pools, mp)
	}

	for _, pg := range ps.Groups {
		mg := &Group{
			Description: &pg.Description,
			Pools:       []string{},
		}

		for _, pp := range pg.Pools {

			if _, ok := pmap[pp]; !ok {
				continue
			}

			mg.Pools = append(mg.Pools, pmap[pp])
		}
		m.Groups = append(m.Groups, *mg)
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

	ps := pool.NewPoolStore().WithSecret("somesecret")

	name := "stuff"

	g0 := pool.NewGroup(name)
	ps.AddGroup(g0)

	g1 := pool.NewGroup("things")
	ps.AddGroup(g1)

	p0 := pool.NewPool("stuff0")

	p0.DisplayInfo = pool.DisplayInfo{
		Short:   "The Good Stuff - Pool 0",
		Long:    "This stuff has some good stuff in it",
		Further: "https://example.com/further.html",
		Thumb:   "https://example.com/thumb.png",
		Image:   "https://example.com/img.png",
	}

	g0.AddPool(p0)
	g1.AddPool(p0)

	ps.AddPool(p0)

	p1 := pool.NewPool("things0")

	p1.DisplayInfo = pool.DisplayInfo{
		Short:   "The Good Things - Pool 0",
		Long:    "This thing has some good things in it",
		Further: "https://example.com/further.html",
		Thumb:   "https://example.com/thumb.png",
		Image:   "https://example.com/img.png",
	}

	g1.AddPool(p1)

	ps.AddPool(p1)

	// Activity A
	a := pool.NewActivity("acti00", ps.Now()+duration)

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

	// Activity B

	b := pool.NewActivity("acti01", ps.Now()+duration)

	p0.AddActivity(b)

	pt2 := permission.Token{
		ConnectionType: "session",
		Topic:          "230498529083",
		Scopes:         []string{"read", "write"},
		StandardClaims: jwt.StandardClaims{
			Audience: "https://example.com",
		},
	}
	s2 := pool.NewStream("https://example.com/session/23049852908")
	s2.SetPermission(pt2)
	b.AddStream("data", s2)

	pt3 := permission.Token{
		ConnectionType: "session",
		Topic:          "asdfasdf456",
		Scopes:         []string{"read"},
		StandardClaims: jwt.StandardClaims{
			Audience: "https://example.com",
		},
	}
	s3 := pool.NewStream("https://example.com/session/asdfasdf456")
	s3.SetPermission(pt3)
	b.AddStream("video", s3)

	b.AddUI(u0)
	b.AddUI(u1)

	return ps

}
*/
