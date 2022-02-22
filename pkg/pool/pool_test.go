package pool

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/timdrysdale/relay/pkg/booking/models"
	"github.com/timdrysdale/relay/pkg/permission"
)

var debug bool

func init() {
	debug = false
	if debug {
		log.SetReportCaller(true)
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&logrus.TextFormatter{FullTimestamp: false, DisableColors: true})
		defer log.SetOutput(os.Stdout)

	} else {
		log.SetLevel(log.WarnLevel)
		var ignore bytes.Buffer
		logignore := bufio.NewWriter(&ignore)
		log.SetOutput(logignore)
	}

}

func mockTime(now *int64) int64 {
	return *now
}

func TestTypeConversionWithNilPointers(t *testing.T) {

	// These test are attempting to provoke
	// nil pointer panics.
	// no panic = pass

	NewActivityFromModel(nil)

	NewActivityFromModel(&models.Activity{})

	ma := &models.Activity{
		Description: nil,
		Exp:         nil,
		Streams:     []*models.Stream{nil, nil},
		Uis:         []*models.UserInterface{nil, nil},
	}
	NewActivityFromModel(ma)

	md := &models.Description{
		Name: nil,
		Type: nil,
	}
	NewDescriptionFromModel(md)

	ms := &models.Stream{
		For:        nil,
		Permission: nil,
		Token:      "",
		URL:        nil,
		Verb:       nil,
	}

	NewSingleStreamFromModel(ms)

	mui := &models.UserInterface{
		Description: nil,
		URL:         nil,
	}
	NewSingleUIFromModel(mui)

}

func TestTypeConversion(t *testing.T) {

	debug := false

	Further := "Further"
	ID := "ID"
	Image := "Image"
	Long := "Long"
	Short := "Short"
	Name := "Name"
	Thumb := "Thumb"
	Type := "Type"

	d := models.Description{
		Further: Further,
		ID:      ID,
		Image:   Image,
		Long:    Long,
		Short:   Short,
		Name:    &Name,
		Thumb:   Thumb,
		Type:    &Type,
	}

	Audience := "Audience"
	ConnectionType := "ConnectionType"
	Scopes := []string{"Sco", "pes"}
	Topic := "Topic"

	p := models.Permission{
		Audience:       &Audience,
		ConnectionType: &ConnectionType,
		Scopes:         Scopes,
		Topic:          &Topic,
	}

	For := "For"
	Token := "Token"
	URL := "URL"
	Verb := "Verb"
	Exp := float64(789)

	s0 := models.Stream{
		For:        &For,
		Permission: &p,
		Token:      Token,
		URL:        &URL,
		Verb:       &Verb,
	}

	s1 := s0
	roF := "roF"
	s1.For = &roF

	StreamsRequired := []string{"Streams", "Required"}

	d1 := d
	d2 := d

	d1.ID = "someUI"
	d2.ID = "anotherUI"

	URL1 := "someURL"
	URL2 := "anotherURL"

	u0 := models.UserInterface{
		Description:     &d1,
		URL:             &URL1,
		StreamsRequired: StreamsRequired,
	}

	u1 := models.UserInterface{
		Description:     &d2,
		URL:             &URL2,
		StreamsRequired: StreamsRequired,
	}

	ma := &models.Activity{
		Description: &d,
		Exp:         &Exp,
		Streams:     []*models.Stream{&s0, &s1},
		Uis:         []*models.UserInterface{&u0, &u1},
	}

	a := NewActivityFromModel(ma)

	if debug {
		pretty, err := json.MarshalIndent(*a, "", "\t")
		assert.NoError(t, err)
		fmt.Println(string(pretty))
	}

	assert.Equal(t, ID, a.ID)
	assert.Equal(t, int64(Exp), a.ExpiresAt)
	assert.Equal(t, 2, len(a.Streams))
	assert.Equal(t, 2, len(a.UI))

	// check streams and UI are unique
	hasFor := false
	hasroF := false

	for key, stream := range a.Streams {
		if key == For {
			hasFor = true
			assert.Equal(t, For, stream.For)
		}
		if key == roF {
			hasroF = true
			assert.Equal(t, roF, stream.For)
		}
	}

	if !(hasFor && hasroF) {
		t.Error("missing one or both streams")
	}

	hasSome := false
	hasAnother := false

	for _, ui := range a.UI {
		if ui.ID == "someUI" {
			hasSome = true
			assert.Equal(t, URL1, ui.URL)
		}
		if ui.ID == "anotherUI" {
			hasAnother = true
			assert.Equal(t, URL2, ui.URL)
		}
	}
	if !(hasSome && hasAnother) {
		t.Error("missing one or both UI")
	}

	ma2 := a.ConvertToModel()
	if debug {
		pretty, err := json.MarshalIndent(*ma2, "", "\t")
		assert.NoError(t, err)
		fmt.Println(string(pretty))
	}

	assert.Equal(t, *ma.Description, *ma2.Description)
	assert.Equal(t, *ma.Exp, *ma2.Exp)
	// UI and Streams could be in any order...

	if *ma.Streams[0].For == *ma2.Streams[0].For {

		assert.Equal(t, *ma.Streams[0], *ma2.Streams[0])
		assert.Equal(t, *ma.Streams[1], *ma2.Streams[1])

	} else {

		assert.Equal(t, *ma.Streams[0], *ma2.Streams[1])
		assert.Equal(t, *ma.Streams[1], *ma2.Streams[0])
	}

	if ma.Uis[0].Description.ID == ma2.Uis[0].Description.ID {

		assert.Equal(t, *ma.Uis[0], *ma2.Uis[0])
		assert.Equal(t, *ma.Uis[1], *ma2.Uis[1])

	} else {

		assert.Equal(t, *ma.Uis[0], *ma2.Uis[1])
		assert.Equal(t, *ma.Uis[1], *ma2.Uis[0])
	}

	pt := MakeClaims(ma.Streams[0].Permission)

	assert.Equal(t, reflect.TypeOf(permission.Token{}), reflect.TypeOf(pt))

	assert.Equal(t, Audience, pt.Audience[0])
	assert.Equal(t, Topic, pt.Topic)
	assert.Equal(t, ConnectionType, pt.ConnectionType)
	assert.Equal(t, Scopes, pt.Scopes)

	// now check the activity checker works ok...

	err := CheckActivity(a)
	assert.Error(t, err)
	expected := fmt.Sprintf("activity already expired at 789 (time now is %d)", time.Now().Unix())
	assert.Equal(t, expected, err.Error())

	later := float64(time.Now().Unix() + 3600)
	ma.Exp = &later
	a = NewActivityFromModel(ma)
	err = CheckActivity(a)
	assert.Error(t, err)
	assert.Equal(t, "audience not an url because parse \"Audience\": invalid URI for request", err.Error())

	audience := "https://example.com"
	ma.Streams[0].Permission.Audience = &audience
	ma.Streams[1].Permission.Audience = &audience
	a = NewActivityFromModel(ma)
	err = CheckActivity(a)
	assert.Error(t, err)
	assert.Equal(t, "connection_type ConnectionType is not session or shell", err.Error())

	session := "session"
	shell := "shell"
	ma.Streams[0].Permission.ConnectionType = &session
	ma.Streams[1].Permission.ConnectionType = &shell
	a = NewActivityFromModel(ma)
	err = CheckActivity(a)
	assert.NoError(t, err)

	// now introduce some other critical errors...
	a.ID = ""
	err = CheckActivity(a)
	assert.Equal(t, "no id", err.Error())

	a.ID = "ljaldskjf09q27843r0982"      //fix that ...
	a.Streams[For].Permission.Topic = "" //break this
	err = CheckActivity(a)
	assert.Equal(t, "empty topic", err.Error())

	a.Streams[For].Permission.Topic = "Topic"               //fix that ...
	a.Streams[For].Permission.Audience = jwt.ClaimStrings{} //break this (by setting no audience)

	err = CheckActivity(a)
	assert.Error(t, err)
	assert.Equal(t, "empty audience", err.Error())

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
	assert.NoError(t, err)

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

	_, err = p.ActivityWaitDuration(5000) //none left, b will be expired before session ends
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

	_, err = p.ActivityRequestAny(1000) // none left!
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

	ps := NewStore().WithSecret("foo")

	assert.Equal(t, []byte("foo"), ps.GetSecret())

	p0 := NewPool("stuff0")
	p1 := NewPool("stuff1")
	p2 := NewPool("things")

	ps.AddPool(p0)
	ps.AddPool(p1)
	ps.AddPool(p2)

	_, err := ps.GetPoolByID("definitelyNotAPoolIDBecauseNotAUUID")

	assert.Error(t, err)

	assert.Equal(t, "not found", err.Error())

	pool, err := ps.GetPoolByID(p0.ID)

	assert.NoError(t, err)

	assert.Equal(t, p0.Name, pool.Name)

	pools := ps.GetAllPools()
	assert.Equal(t, 3, len(pools))

	ids := ps.GetAllPoolIDs()
	assert.Equal(t, 3, len(ids))

	pools, err = ps.GetPoolsByName("stuff1")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pools))

	assert.Equal(t, p1.Name, (pools[0]).Name)

	pools, err = ps.GetPoolsByNamePrefix("stuff")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(pools))

	ps.DeletePool(p0)
	pools, err = ps.GetPoolsByNamePrefix("stuff")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pools))
	ps.DeletePool(p1)
	_, err = ps.GetPoolsByNamePrefix("stuff")
	assert.Error(t, err)
	assert.Equal(t, "not found", err.Error())

}

func TestAddGetDeleteGroups(t *testing.T) {

	ps := NewStore().WithSecret("bar")

	assert.Equal(t, []byte("bar"), ps.GetSecret())

	g0 := NewGroup("stuff0")
	g1 := NewGroup("stuff1")
	g2 := NewGroup("things")

	ps.AddGroup(g0)
	ps.AddGroup(g1)
	ps.AddGroup(g2)

	_, err := ps.GetGroupByID("definitelyNotAGroupIDBecauseNotAUUID")

	assert.Error(t, err)

	assert.Equal(t, "not found", err.Error())

	group, err := ps.GetGroupByID(g0.ID)

	assert.NoError(t, err)

	assert.Equal(t, g0.Name, group.Name)

	groups, err := ps.GetGroupsByName("stuff1")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(groups))

	assert.Equal(t, g1.Name, (groups[0]).Name)

	groups, err = ps.GetGroupsByNamePrefix("stuff")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(groups))

	ps.DeleteGroup(g0)

	groups, err = ps.GetGroupsByNamePrefix("stuff")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(groups))

	ps.DeleteGroup(g1)

	_, err = ps.GetGroupsByNamePrefix("stuff")
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

func TestAddPermissionsToStream(t *testing.T) {

	p := permission.Token{
		ConnectionType: "session",
		Topic:          "123",
		RegisteredClaims: jwt.RegisteredClaims{
			Audience: jwt.ClaimStrings{"https://example.com"},
		},
	}

	s := NewStream("https://example.com/some/stream").WithPermission(p)

	assert.Equal(t, p, s.GetPermission())

}

func TestImportExport(t *testing.T) {

	// *** Setup groups, pools, activities *** //

	mocktime := time.Now().Unix()

	ps := NewStore().WithSecret("bar")

	name := "stuff"
	g0 := NewGroup(name)
	ps.AddGroup(g0)
	defer ps.DeleteGroup(g0)

	p0 := NewPool("stuff0").WithNow(func() int64 { return func(now *int64) int64 { return *now }(&mocktime) })
	g0.AddPool(p0)
	ps.AddPool(p0)
	defer ps.DeletePool(p0)

	a := NewActivity("a", ps.Now()+3600)

	err := p0.AddActivity(a)
	assert.NoError(t, err)

	defer p0.DeleteActivity(a)

	pt0 := permission.Token{
		ConnectionType: "session",
		Topic:          "foo",
		Scopes:         []string{"read", "write"},
		RegisteredClaims: jwt.RegisteredClaims{
			Audience: jwt.ClaimStrings{"https://example.com"},
		},
	}
	s0 := NewStream("https://example.com/session/123data")
	s0.SetPermission(pt0)
	a.AddStream("data", s0)

	pt1 := permission.Token{
		ConnectionType: "session",
		Topic:          "foo", //would not normally set same as other stream - testing convenience
		Scopes:         []string{"read"},
		RegisteredClaims: jwt.RegisteredClaims{
			Audience: jwt.ClaimStrings{"https://example.com"},
		},
	}
	s1 := NewStream("https://example.com/session/456video")
	s1.SetPermission(pt1)
	a.AddStream("video", s1)

	a2 := NewActivity("a2", ps.Now()+3600)
	err = p0.AddActivity(a2)
	assert.NoError(t, err)
	defer p0.DeleteActivity(a2)

	pt2 := permission.Token{
		ConnectionType: "session",
		Topic:          "bar",
		Scopes:         []string{"read", "write"},
		RegisteredClaims: jwt.RegisteredClaims{
			Audience: jwt.ClaimStrings{"https://example.com"},
		},
	}

	s2 := NewStream("https://example.com/session/123data")
	s2.SetPermission(pt2)
	a2.AddStream("data", s2)

	pt3 := permission.Token{
		ConnectionType: "session",
		Topic:          "bar", //would not normally set same as other stream - testing convenience
		Scopes:         []string{"read"},
		RegisteredClaims: jwt.RegisteredClaims{
			Audience: jwt.ClaimStrings{"https://example.com"},
		},
	}
	s3 := NewStream("https://example.com/session/456video")
	s3.SetPermission(pt3)
	a2.AddStream("video", s3)

	b, err := ps.ExportAll()

	assert.NoError(t, err)

	// kill the poolstore
	ps = &Store{
		RWMutex: &sync.RWMutex{},
	}

	// check it's dead
	_, err = ps.GetPoolByID(p0.ID)
	assert.Error(t, err)

	// restore the poolstore
	ps2, err := ImportAll(b)
	assert.NoError(t, err)

	ps2.PostImportSetNow(func() int64 { return func(now *int64) int64 { return *now }(&mocktime) })
	ps = ps2

	// *** run the rest of the test *** //

	mocktime = time.Now().Unix()

	p, err := ps.GetPoolByID(p0.ID)

	assert.NoError(t, err)

	aID0, err := p.ActivityRequestAny(2000)
	assert.NoError(t, err)

	agot0, err := p.GetActivityByID(aID0)
	assert.NoError(t, err)
	// now request a second activity from the same user ...
	aID1, err := p.ActivityRequestAny(2000)
	assert.NoError(t, err)
	agot1, err := p.GetActivityByID(aID1)
	assert.NoError(t, err)
	topic0 := agot0.Streams["data"].Permission.Topic
	topic1 := agot1.Streams["data"].Permission.Topic

	//'123' is from activity 'a'; '789' is from activity 'a2'
	if !((topic0 == "foo" && topic1 == "bar") || (topic0 == "bar" && topic1 == "foo")) {
		t.Error("didn't get the right permission tokens - did we get the same activity twice?")
	}

}
