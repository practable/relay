package book

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/phayes/freeport"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/timdrysdale/relay/pkg/booking"
	"github.com/timdrysdale/relay/pkg/booking/models"
	"github.com/timdrysdale/relay/pkg/bookingstore"
	lit "github.com/timdrysdale/relay/pkg/login"
	"github.com/timdrysdale/relay/pkg/permission"
	"github.com/timdrysdale/relay/pkg/pool"
	"github.com/timdrysdale/relay/pkg/util"
)

var l *bookingstore.Limit
var ps *pool.PoolStore
var host, secret string
var bookingDuration, mocktime, startime int64

// Deferred deletes are to clean up between tests
// and are not an example of how to use the system
// in production - you want the items to live on
// so that some booking can be done!

func init() {
	debug := false
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

func TestMain(m *testing.M) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	secret = "somesecret"
	bookingDuration = int64(180)

	mocktime = time.Now().Unix()
	startime = mocktime

	ps = pool.NewPoolStore().
		WithSecret(secret).
		WithBookingTokenDuration(bookingDuration).
		WithNow(func() int64 { return func(now *int64) int64 { return *now }(&mocktime) })

	l = bookingstore.New(ctx).WithFlush(time.Minute).WithMax(2).WithProvisionalPeriod(5 * time.Second)

	port, err := freeport.GetFreePort()
	if err != nil {
		panic(err)
	}

	host = "http://[::]:" + strconv.Itoa(port)

	go booking.API(ctx, port, host, secret, ps, l)

	time.Sleep(time.Second)

	exitVal := m.Run()

	os.Exit(exitVal)
}

func TestBooking(t *testing.T) {

	loginClaims := &lit.Token{}
	loginClaims.Audience = host
	loginClaims.Groups = []string{"somecourse", "everyone"}
	loginClaims.Scopes = []string{"login:user"}
	loginClaims.IssuedAt = ps.GetTime() - 1
	loginClaims.NotBefore = ps.GetTime() - 1
	loginClaims.ExpiresAt = loginClaims.NotBefore + ps.BookingTokenDuration
	// sign user token
	loginToken := jwt.NewWithClaims(jwt.SigningMethodHS256, loginClaims)
	// Sign and get the complete encoded token as a string using the secret
	loginBearer, err := loginToken.SignedString([]byte(ps.Secret))
	assert.NoError(t, err)

	client := &http.Client{}

	req, err := http.NewRequest("POST", host+"/api/v1/login", nil)
	assert.NoError(t, err)
	resp, err := client.Do(req)
	assert.NoError(t, err)

	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	bodyStr := string([]byte(body))
	assert.Equal(t, `{"code":401,"message":"unauthenticated for invalid credentials"}`, bodyStr)

	req, err = http.NewRequest("POST", host+"/api/v1/login", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", loginBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	btr := &models.Bookingtoken{}
	err = json.Unmarshal(body, btr)
	assert.NoError(t, err)

	if btr == nil {
		t.Fatal("no token returned")
	}
	bookingTokenReturned := *(btr.Token)

	token, err := jwt.ParseWithClaims(bookingTokenReturned, &lit.Token{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)

	claims, ok := token.Claims.(*lit.Token)

	assert.True(t, ok)
	assert.True(t, token.Valid)

	assert.Equal(t, []string{"somecourse", "everyone"}, claims.Groups)
	assert.Equal(t, []string{"booking:user"}, claims.Scopes)
	assert.True(t, claims.ExpiresAt < ps.Now()+bookingDuration+15)
	assert.True(t, claims.ExpiresAt > ps.Now()+bookingDuration-15)
	assert.True(t, len(claims.Subject) >= 35)

	subject := claims.Subject //save for next test

	// Now login again with booking token in body and see that subject is retained
	respBody, err := json.Marshal(lit.TokenInBody{Token: bookingTokenReturned})
	assert.NoError(t, err)
	req, err = http.NewRequest("POST", host+"/api/v1/login", bytes.NewBuffer(respBody))
	assert.NoError(t, err)
	req.Header.Set("Content-type", "application/json")
	req.Header.Add("Authorization", loginBearer)
	resp, err = client.Do(req)

	assert.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	btr = &models.Bookingtoken{}

	err = json.Unmarshal(body, btr)
	assert.NoError(t, err)

	token, err = jwt.ParseWithClaims(*(btr.Token), &lit.Token{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)

	claims, ok = token.Claims.(*lit.Token)

	assert.True(t, ok)
	assert.True(t, token.Valid)

	assert.Equal(t, []string{"somecourse", "everyone"}, claims.Groups)
	assert.Equal(t, []string{"booking:user"}, claims.Scopes)
	assert.True(t, claims.ExpiresAt < ps.Now()+bookingDuration+15)
	assert.True(t, claims.ExpiresAt > ps.Now()+bookingDuration-15)
	assert.True(t, len(claims.Subject) >= 35)

	// key test
	assert.Equal(t, subject, claims.Subject)

}

func TestGetGroupIDByName(t *testing.T) {

	g0 := pool.NewGroup("stuff")
	ps.AddGroup(g0)
	defer ps.DeleteGroup(g0)

	g1 := pool.NewGroup("things")
	ps.AddGroup(g1)
	defer ps.DeleteGroup(g1)

	claims := &lit.Token{}
	claims.Audience = host
	claims.Groups = []string{"stuff"}
	claims.Scopes = []string{"booking:user"}
	claims.IssuedAt = ps.GetTime() - 1
	claims.NotBefore = ps.GetTime() - 1
	claims.ExpiresAt = claims.NotBefore + ps.BookingTokenDuration

	// sign user token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Sign and get the complete encoded token as a string using the secret
	bearer, err := token.SignedString([]byte(ps.Secret))
	assert.NoError(t, err)

	client := &http.Client{}
	req, err := http.NewRequest("GET", host+"/api/v1/groups", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)
	q := req.URL.Query()
	q.Add("name", "stuff")
	req.URL.RawQuery = q.Encode()
	resp, err := client.Do(req)
	assert.NoError(t, err)
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	ids := []string{}
	err = json.Unmarshal(body, &ids)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ids))
	assert.Equal(t, g0.ID, ids[0])

	// check request fails if group not in groups
	req, err = http.NewRequest("GET", host+"/api/v1/groups", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)
	q = req.URL.Query()
	q.Add("name", "things")
	req.URL.RawQuery = q.Encode()
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "\"Missing Group in Groups Claim\"\n", string(body))

}

func TestGetGroupDescriptionByID(t *testing.T) {

	name := "stuff"

	g0 := pool.NewGroup(name)

	g0.DisplayInfo = pool.DisplayInfo{
		Short:   "Some Good Stuff",
		Long:    "This stuff has some good stuff in it",
		Further: "https://example.com/further.html",
		Thumb:   "https://example.com/thumb.png",
		Image:   "https://example.com/img.png",
	}

	ps.AddGroup(g0)
	defer ps.DeleteGroup(g0)

	claims := &lit.Token{}
	claims.Audience = host
	claims.Groups = []string{name}
	claims.Scopes = []string{"booking:user"}
	claims.IssuedAt = ps.GetTime() - 1
	claims.NotBefore = ps.GetTime() - 1
	claims.ExpiresAt = claims.NotBefore + ps.BookingTokenDuration

	// sign user token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Sign and get the complete encoded token as a string using the secret
	bearer, err := token.SignedString([]byte(ps.Secret))
	assert.NoError(t, err)

	client := &http.Client{}
	req, err := http.NewRequest("GET", host+"/api/v1/groups/"+g0.ID, nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)
	assert.NoError(t, err)
	resp, err := client.Do(req)
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	d := models.Description{}
	err = json.Unmarshal(body, &d)
	assert.NoError(t, err)
	assert.Equal(t, "Some Good Stuff", d.Short)
	assert.Equal(t, "This stuff has some good stuff in it", d.Long)
	assert.Equal(t, "https://example.com/further.html", d.Further)
	assert.Equal(t, "https://example.com/thumb.png", d.Thumb)
	assert.Equal(t, "https://example.com/img.png", d.Image)
	assert.Equal(t, g0.ID, d.ID)

}

func TestGetAllPools(t *testing.T) {

	name := "stuff"

	g0 := pool.NewGroup(name)
	defer ps.DeleteGroup(g0)

	p0 := pool.NewPool("stuff0")
	ps.AddPool(p0)
	defer ps.DeletePool(p0)

	p1 := pool.NewPool("stuff1")
	ps.AddPool(p1)
	defer ps.DeletePool(p1)

	p2 := pool.NewPool("things")
	ps.AddPool(p2)
	defer ps.DeletePool(p2)

	p3 := pool.NewPool("stuff00")
	ps.AddPool(p3)
	defer ps.DeletePool(p3)

	// groups don't affect this test
	// leaving this here as a reminder
	// you have to add pools to poolstore
	// separately to adding to group.
	ps.AddGroup(g0)
	defer ps.DeleteGroup(g0)

	g0.AddPools([]*pool.Pool{p0, p1})

	//check pools exist directly
	ps.Lock()
	assert.Equal(t, 4, len(ps.Pools))
	ps.Unlock()

	claims := &lit.Token{}
	claims.Audience = host
	claims.Groups = []string{name}
	claims.Scopes = []string{"booking:admin"}
	claims.IssuedAt = ps.GetTime() - 1
	claims.NotBefore = ps.GetTime() - 1
	claims.ExpiresAt = claims.NotBefore + ps.BookingTokenDuration

	// sign user token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Sign and get the complete encoded token as a string using the secret
	bearer, err := token.SignedString([]byte(ps.Secret))
	assert.NoError(t, err)

	client := &http.Client{}
	req, err := http.NewRequest("GET", host+"/api/v1/pools/", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)
	assert.NoError(t, err)
	resp, err := client.Do(req)
	body, err := ioutil.ReadAll(resp.Body)
	if false {
		t.Log(resp.Status, string(body))
	}
	assert.NoError(t, err)

	r := []string{}

	err = json.Unmarshal(body, &r)
	assert.NoError(t, err)

	// note the order can change in PoolStore - that's ok
	assert.True(t, util.SortCompare([]string{p0.ID, p1.ID, p2.ID, p3.ID}, r))

	req, err = http.NewRequest("GET", host+"/api/v1/pools/", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)
	assert.NoError(t, err)
	q := req.URL.Query()
	q.Add("name", "stuff0")
	req.URL.RawQuery = q.Encode()
	resp, err = client.Do(req)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	r = []string{}
	err = json.Unmarshal(body, &r)
	assert.NoError(t, err)
	assert.True(t, util.SortCompare([]string{p0.ID, p3.ID}, r))

	req, err = http.NewRequest("GET", host+"/api/v1/pools/", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bearer)
	assert.NoError(t, err)
	q = req.URL.Query()
	q.Add("name", "stuff0")
	q.Add("exact", "true")
	req.URL.RawQuery = q.Encode()
	resp, err = client.Do(req)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	r = []string{}
	err = json.Unmarshal(body, &r)
	assert.NoError(t, err)
	assert.Equal(t, []string{p0.ID}, r)

}

func TestGetPoolsAtLoginDescriptionStatusByID(t *testing.T) {

	// add groups, pools

	name := "stuff"
	g0 := pool.NewGroup(name)
	defer ps.DeleteGroup(g0)

	ps.AddGroup(g0)
	defer ps.DeleteGroup(g0)

	p0 := pool.NewPool("stuff0").WithNow(func() int64 { return func(now *int64) int64 { return *now }(&mocktime) })

	p0.DisplayInfo = pool.DisplayInfo{
		Short:   "The Good Stuff - Pool 0",
		Long:    "This stuff has some good stuff in it",
		Further: "https://example.com/further.html",
		Thumb:   "https://example.com/thumb.png",
		Image:   "https://example.com/img.png",
	}

	p1 := pool.NewPool("stuff1")
	g0.AddPools([]*pool.Pool{p0, p1})

	ps.AddPool(p0)
	defer ps.DeletePool(p0)

	ps.AddPool(p1)
	defer ps.DeletePool(p1)

	// login
	loginClaims := &lit.Token{}
	loginClaims.Audience = host
	//check that missing group "everyone" in PoolStore does not stop login
	loginClaims.Groups = []string{name, "everyone"}
	loginClaims.Scopes = []string{"login:user"}
	loginClaims.IssuedAt = ps.GetTime() - 1
	loginClaims.NotBefore = ps.GetTime() - 1
	loginClaims.ExpiresAt = loginClaims.NotBefore + ps.BookingTokenDuration
	// sign user token
	loginToken := jwt.NewWithClaims(jwt.SigningMethodHS256, loginClaims)
	// Sign and get the complete encoded token as a string using the secret
	loginBearer, err := loginToken.SignedString([]byte(ps.Secret))
	assert.NoError(t, err)

	client := &http.Client{}

	req, err := http.NewRequest("POST", host+"/api/v1/login", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", loginBearer)
	resp, err := client.Do(req)
	assert.NoError(t, err)

	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	btr := &models.Bookingtoken{}
	err = json.Unmarshal(body, btr)
	assert.NoError(t, err)

	if btr == nil {
		t.Fatal("no token returned")
	}

	bookingBearer := *(btr.Token)

	token, err := jwt.ParseWithClaims(bookingBearer, &lit.Token{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)

	claims, ok := token.Claims.(*lit.Token)

	assert.True(t, ok)
	assert.True(t, token.Valid)

	assert.Equal(t, []string{p0.ID, p1.ID}, claims.Pools)

	// Check we can get a description of a pool
	req, err = http.NewRequest("GET", host+"/api/v1/pools/"+p0.ID, nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bookingBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	d := models.Description{}
	err = json.Unmarshal(body, &d)
	assert.NoError(t, err)
	assert.Equal(t, "The Good Stuff - Pool 0", d.Short)
	assert.Equal(t, "This stuff has some good stuff in it", d.Long)
	assert.Equal(t, "https://example.com/further.html", d.Further)
	assert.Equal(t, "https://example.com/thumb.png", d.Thumb)
	assert.Equal(t, "https://example.com/img.png", d.Image)
	assert.Equal(t, p0.ID, d.ID)

	// get status with no activities registered to check this doesn't break anything
	req, err = http.NewRequest("GET", host+"/api/v1/pools/"+p0.ID+"/status", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bookingBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	s := models.Status{}
	err = json.Unmarshal(body, &s)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), *s.Available)
	assert.Equal(t, int64(0), s.Wait)
	assert.Equal(t, false, s.Later)
	assert.Equal(t, int64(0), s.Used)

	// Add some activities
	a := pool.NewActivity("a", ps.Now()+3600)
	b := pool.NewActivity("b", ps.Now()+7200)
	c := pool.NewActivity("b", ps.Now()+7200)
	p0.AddActivity(a)
	defer p0.DeleteActivity(a)
	p0.AddActivity(b)
	defer p0.DeleteActivity(b)
	p0.AddActivity(c)
	defer p0.DeleteActivity(c)

	req, err = http.NewRequest("GET", host+"/api/v1/pools/"+p0.ID+"/status", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bookingBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	err = json.Unmarshal(body, &s)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), *s.Available)
	assert.Equal(t, int64(0), s.Wait)
	assert.Equal(t, true, s.Later)
	assert.Equal(t, int64(0), s.Used)

	aid, err := p0.ActivityRequestAny(2000)
	assert.NoError(t, err)
	assert.True(t, a.ID == aid || b.ID == aid || c.ID == aid)

	req, err = http.NewRequest("GET", host+"/api/v1/pools/"+p0.ID+"/status", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bookingBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	err = json.Unmarshal(body, &s)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), *s.Available)
	assert.Equal(t, int64(0), s.Wait)
	assert.Equal(t, true, s.Later)
	assert.Equal(t, int64(1), s.Used)

	aid, err = p0.ActivityRequestAny(2000)
	assert.NoError(t, err)
	assert.True(t, a.ID == aid || b.ID == aid || c.ID == aid)

	aid, err = p0.ActivityRequestAny(2000)
	assert.NoError(t, err)
	assert.True(t, a.ID == aid || b.ID == aid || c.ID == aid)

	req, err = http.NewRequest("GET", host+"/api/v1/pools/"+p0.ID+"/status", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bookingBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	err = json.Unmarshal(body, &s)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), *s.Available)
	assert.Equal(t, int64(2000), s.Wait)
	assert.Equal(t, true, s.Later)
	assert.Equal(t, int64(3), s.Used)

	mocktime = startime + 2002
	assert.Equal(t, mocktime, ps.GetTime())

	req, err = http.NewRequest("GET", host+"/api/v1/pools/"+p0.ID+"/status", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bookingBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	s = models.Status{}
	err = json.Unmarshal(body, &s)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), *s.Available)
	assert.Equal(t, int64(0), s.Wait)
	assert.Equal(t, true, s.Later)
	assert.Equal(t, int64(0), s.Used)

	mocktime = startime + 3601

	aid, err = p0.ActivityRequestAny(2000)
	assert.NoError(t, err)
	assert.True(t, b.ID == aid || c.ID == aid)

	aid, err = p0.ActivityRequestAny(2000)
	assert.NoError(t, err)
	assert.True(t, b.ID == aid || c.ID == aid)

	req, err = http.NewRequest("GET", host+"/api/v1/pools/"+p0.ID+"/status", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bookingBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	s = models.Status{}
	err = json.Unmarshal(body, &s)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), *s.Available)
	assert.Equal(t, int64(2000), s.Wait)
	assert.Equal(t, true, s.Later)
	assert.Equal(t, int64(2), s.Used)

	// now try again but checking status for a longer requested duration
	// which is longer than they are available
	// become available at 5601, expire at 7200

	mocktime = startime + 5300

	req, err = http.NewRequest("GET", host+"/api/v1/pools/"+p0.ID+"/status", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bookingBearer)
	q := req.URL.Query()
	q.Add("duration", "2000")
	req.URL.RawQuery = q.Encode()
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	s = models.Status{}
	err = json.Unmarshal(body, &s)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), *s.Available)
	assert.Equal(t, int64(0), s.Wait)
	assert.Equal(t, false, s.Later)
	assert.Equal(t, int64(2), s.Used)

}

func TestRequestSessionByPoolID(t *testing.T) {

	name := "stuff"
	g0 := pool.NewGroup(name)
	defer ps.DeleteGroup(g0)

	ps.AddGroup(g0)
	defer ps.DeleteGroup(g0)

	p0 := pool.NewPool("stuff0").WithNow(func() int64 { return func(now *int64) int64 { return *now }(&mocktime) })

	p0.DisplayInfo = pool.DisplayInfo{
		Short:   "The Good Stuff - Pool 0",
		Long:    "This stuff has some good stuff in it",
		Further: "https://example.com/further.html",
		Thumb:   "https://example.com/thumb.png",
		Image:   "https://example.com/img.png",
	}

	g0.AddPool(p0)
	ps.AddPool(p0)
	defer ps.DeletePool(p0)

	a := pool.NewActivity("a", ps.Now()+3600)

	p0.AddActivity(a)
	defer p0.DeleteActivity(a)

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

	mocktime = time.Now().Unix()

	// login
	loginClaims := &lit.Token{}
	loginClaims.Audience = host
	//check that missing group "everyone" in PoolStore does not stop login
	loginClaims.Groups = []string{name, "everyone"}
	loginClaims.Scopes = []string{"login:user"}
	loginClaims.IssuedAt = ps.GetTime() - 1
	loginClaims.NotBefore = ps.GetTime() - 1
	loginClaims.ExpiresAt = loginClaims.NotBefore + ps.BookingTokenDuration
	// sign user token
	loginToken := jwt.NewWithClaims(jwt.SigningMethodHS256, loginClaims)
	// Sign and get the complete encoded token as a string using the secret
	loginBearer, err := loginToken.SignedString([]byte(ps.Secret))
	assert.NoError(t, err)

	mocktime = time.Now().Unix()

	client := &http.Client{}
	req, err := http.NewRequest("POST", host+"/api/v1/login", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", loginBearer)
	resp, err := client.Do(req)
	assert.NoError(t, err)

	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	btr := &models.Bookingtoken{}
	err = json.Unmarshal(body, btr)
	assert.NoError(t, err)

	if btr == nil {
		t.Fatal("no token returned")
	}

	bookingBearer := *(btr.Token)

	token, err := jwt.ParseWithClaims(bookingBearer, &lit.Token{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)

	claims, ok := token.Claims.(*lit.Token)

	assert.True(t, ok)
	assert.True(t, token.Valid)

	assert.Equal(t, []string{p0.ID}, claims.Pools)

	// request an activity...
	req, err = http.NewRequest("POST", host+"/api/v1/pools/"+p0.ID+"/sessions", nil)
	assert.NoError(t, err)
	q := req.URL.Query()
	q.Add("duration", "2000")
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Authorization", bookingBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	ma := &models.Activity{}
	err = json.Unmarshal(body, ma)
	assert.NoError(t, err)

	if ma == nil {
		t.Fatal("no token returned")
	}

	streamTokenString0 := *((ma.Streams[0]).Token)
	streamTokenString1 := *((ma.Streams[1]).Token)

	assert.Equal(t, "a", *(ma.Description.Name))
	assert.Equal(t, 2, len(ma.Streams))
	assert.Equal(t, 2, len(ma.Uis))
	assert.Equal(t, "ey", (*((ma.Streams[0]).Token))[0:2])
	assert.Equal(t, "ey", streamTokenString1[0:2])

	ptclaims := &permission.Token{}

	streamToken, err := jwt.ParseWithClaims(streamTokenString0, ptclaims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method was %v", token.Header["alg"])
		}
		return []byte(ps.Secret), nil
	})

	if err != nil {
		t.Fatal(err)
	}

	stc, ok := streamToken.Claims.(*permission.Token)

	assert.Equal(t, pt0.Audience, stc.Audience)
	assert.Equal(t, pt0.ConnectionType, stc.ConnectionType)
	assert.True(t, pt0.Topic == stc.Topic || pt1.Topic == stc.Topic)
	assert.Equal(t, ps.Now()+2000, stc.ExpiresAt)

	// streams could come in either order, so check each item matches one or other
	// TODO: improve this test so it detects weird mistakes like mixing up stream data
	url0 := "https://example.com/session/123data"
	url1 := "https://example.com/session/456video"
	got0 := *(ma.Streams[0].URL)
	got1 := *(ma.Streams[1].URL)

	urlmatch := (url0 == got0 && url1 == got1) || (url0 == got1 && url1 == got0)
	assert.True(t, urlmatch)
	uiurl := "https://static.example.com/example.html?data={{data}}\u0026video={{video}}"
	uiurlmatch := *(ma.Uis[0].URL) == uiurl || *(ma.Uis[1].URL) == uiurl
	assert.True(t, uiurlmatch)

	scopematch := reflect.DeepEqual(ma.Uis[0].StreamsRequired, []string{"data", "video"}) ||
		reflect.DeepEqual(ma.Uis[0].StreamsRequired, []string{"data"})
	assert.True(t, scopematch)

}

func TestLimits(t *testing.T) {
	// minimal activity just for testing- less complete than you'd need in production
	// note that stream order and activity order are not guaranteed - hence the
	// conveniences taken in this test (which is checking limits, not token formation)

	statusCodes := []int{}

	name := "stuff"
	g0 := pool.NewGroup(name)
	defer ps.DeleteGroup(g0)

	ps.AddGroup(g0)
	defer ps.DeleteGroup(g0)

	p0 := pool.NewPool("stuff0").WithNow(func() int64 { return func(now *int64) int64 { return *now }(&mocktime) })

	g0.AddPool(p0)
	ps.AddPool(p0)
	defer ps.DeletePool(p0)

	a := pool.NewActivity("a", ps.Now()+3600)

	p0.AddActivity(a)
	defer p0.DeleteActivity(a)

	pt0 := permission.Token{
		ConnectionType: "session",
		Topic:          "foo",
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
		Topic:          "foo", //would not normally set same as other stream - testing convenience
		Scopes:         []string{"read"},
		StandardClaims: jwt.StandardClaims{
			Audience: "https://example.com",
		},
	}
	s1 := pool.NewStream("https://example.com/session/456video")
	s1.SetPermission(pt1)
	a.AddStream("video", s1)

	a2 := pool.NewActivity("a2", ps.Now()+3600)
	p0.AddActivity(a2)
	defer p0.DeleteActivity(a2)

	pt2 := permission.Token{
		ConnectionType: "session",
		Topic:          "bar",
		Scopes:         []string{"read", "write"},
		StandardClaims: jwt.StandardClaims{
			Audience: "https://example.com",
		},
	}

	s2 := pool.NewStream("https://example.com/session/123data")
	s2.SetPermission(pt2)
	a2.AddStream("data", s2)

	pt3 := permission.Token{
		ConnectionType: "session",
		Topic:          "bar", //would not normally set same as other stream - testing convenience
		Scopes:         []string{"read"},
		StandardClaims: jwt.StandardClaims{
			Audience: "https://example.com",
		},
	}
	s3 := pool.NewStream("https://example.com/session/456video")
	s3.SetPermission(pt3)
	a2.AddStream("video", s3)

	mocktime = time.Now().Unix()

	// login
	loginClaims := &lit.Token{}
	loginClaims.Audience = host
	//check that missing group "everyone" in PoolStore does not stop login
	loginClaims.Groups = []string{name, "everyone"}
	loginClaims.Scopes = []string{"login:user"}
	loginClaims.IssuedAt = ps.GetTime() - 1
	loginClaims.NotBefore = ps.GetTime() - 1
	loginClaims.ExpiresAt = loginClaims.NotBefore + ps.BookingTokenDuration
	// sign user token
	loginToken := jwt.NewWithClaims(jwt.SigningMethodHS256, loginClaims)
	// Sign and get the complete encoded token as a string using the secret
	loginBearer, err := loginToken.SignedString([]byte(ps.Secret))
	assert.NoError(t, err)

	mocktime = time.Now().Unix()

	client := &http.Client{}
	req, err := http.NewRequest("POST", host+"/api/v1/login", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", loginBearer)
	resp, err := client.Do(req)
	assert.NoError(t, err)

	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	btr := &models.Bookingtoken{}
	err = json.Unmarshal(body, btr)
	assert.NoError(t, err)

	if btr == nil {
		t.Fatal("no token returned")
	}

	bookingBearer := *(btr.Token)

	token, err := jwt.ParseWithClaims(bookingBearer, &lit.Token{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)

	claims, ok := token.Claims.(*lit.Token)

	assert.True(t, ok)
	assert.True(t, token.Valid)

	assert.Equal(t, []string{p0.ID}, claims.Pools)

	// request an activity...
	req, err = http.NewRequest("POST", host+"/api/v1/pools/"+p0.ID+"/sessions", nil)
	assert.NoError(t, err)
	q := req.URL.Query()
	q.Add("duration", "2000")
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Authorization", bookingBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	ma := &models.Activity{}
	err = json.Unmarshal(body, ma)
	assert.NoError(t, err)

	if ma == nil {
		t.Fatal("no token returned")
	}

	streamTokenString0 := *((ma.Streams[0]).Token)

	ptclaims := &permission.Token{}

	streamToken, err := jwt.ParseWithClaims(streamTokenString0, ptclaims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method was %v", token.Header["alg"])
		}
		return []byte(ps.Secret), nil
	})

	if err != nil {
		t.Fatal(err)
	}

	stc, ok := streamToken.Claims.(*permission.Token)

	// save this to check we get both activities (check data stream permission topic from each request)
	stcTopic0 := stc.Topic

	// now request a second activity from the same user ...
	req, err = http.NewRequest("POST", host+"/api/v1/pools/"+p0.ID+"/sessions", nil)
	assert.NoError(t, err)
	q = req.URL.Query()
	q.Add("duration", "2000")
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Authorization", bookingBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	statusCodes = append(statusCodes, resp.StatusCode)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	ma = &models.Activity{}
	err = json.Unmarshal(body, ma)
	assert.NoError(t, err)

	if ma == nil {
		t.Fatal("no token returned")
	}

	streamTokenString0 = *((ma.Streams[0]).Token)

	ptclaims = &permission.Token{}

	streamToken, err = jwt.ParseWithClaims(streamTokenString0, ptclaims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method was %v", token.Header["alg"])
		}
		return []byte(ps.Secret), nil
	})

	if err != nil {
		t.Fatal(err)
	}

	stc, ok = streamToken.Claims.(*permission.Token)

	// just check the two topics are what we expect from the data permission tokens
	stcTopic1 := stc.Topic

	//'123' is from activity 'a'; '789' is from activity 'a2'
	if !((stcTopic0 == "foo" && stcTopic1 == "bar") || (stcTopic0 == "bar" && stcTopic1 == "foo")) {
		t.Error("didn't get the right permission tokens - did we get the same activity twice?")
	}

	// Now let's try being a different user - we should get a 404 not found (no kit left)
	// by logging in again we'll get a different randomly assigned user id
	req, err = http.NewRequest("POST", host+"/api/v1/login", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", loginBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	statusCodes = append(statusCodes, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	btr2 := &models.Bookingtoken{}
	err = json.Unmarshal(body, btr2)
	assert.NoError(t, err)

	if btr2 == nil {
		t.Fatal("no token returned")
	}

	bookingBearer2 := *(btr2.Token)

	token2, err := jwt.ParseWithClaims(bookingBearer2, &lit.Token{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)

	claims2, ok := token2.Claims.(*lit.Token)

	assert.True(t, ok)
	assert.True(t, token.Valid)

	assert.Equal(t, []string{p0.ID}, claims2.Pools)

	// check not same user - important for next test...
	assert.NotEqual(t, claims.Subject, claims2.Subject)

	// Make the request for the kit ...
	// now request a second activity from the same user ...
	req, err = http.NewRequest("POST", host+"/api/v1/pools/"+p0.ID+"/sessions", nil)
	assert.NoError(t, err)
	q = req.URL.Query()
	q.Add("duration", "2000")
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Authorization", bookingBearer2) //different user this time
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "\"none available\"\n", string(body))
	statusCodes = append(statusCodes, resp.StatusCode)
	// Now let's try being first user again - we should 402 payment required (reached quota)
	req, err = http.NewRequest("POST", host+"/api/v1/pools/"+p0.ID+"/sessions", nil)
	assert.NoError(t, err)
	q = req.URL.Query()
	q.Add("duration", "2000")
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Authorization", bookingBearer) // back to first user this time
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusPaymentRequired, resp.StatusCode)
	assert.Equal(t, "\"Maximum conconcurrent sessions already reached. Try again later.\"\n", string(body))
	statusCodes = append(statusCodes, resp.StatusCode)
	assert.Equal(t, []int{200, 200, 404, 402}, statusCodes)
}

func TestAddNewPool(t *testing.T) {

	// make an admin user login token, swap for booking token
	loginClaims := &lit.Token{}
	loginClaims.Audience = host
	loginClaims.Groups = []string{"everything"} //not an actual group
	loginClaims.Scopes = []string{"login:admin"}
	loginClaims.IssuedAt = ps.GetTime() - 1
	loginClaims.NotBefore = ps.GetTime() - 1
	loginClaims.ExpiresAt = loginClaims.NotBefore + ps.BookingTokenDuration
	loginToken := jwt.NewWithClaims(jwt.SigningMethodHS256, loginClaims)
	loginBearer, err := loginToken.SignedString([]byte(ps.Secret))
	assert.NoError(t, err)

	client := &http.Client{}
	req, err := http.NewRequest("POST", host+"/api/v1/login", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", loginBearer)
	resp, err := client.Do(req)
	assert.NoError(t, err)
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	btr := &models.Bookingtoken{}
	err = json.Unmarshal(body, btr)
	assert.NoError(t, err)

	if btr == nil {
		t.Fatal("no token returned")
	}
	adminBearer := *(btr.Token)

	// make a description, post in body

	further := "https://example.io/further.html"
	image := "https://example.io/image.png"
	long := "some long long long description"
	name := "red"
	short := "short story"
	thumb := "https://example.io/thumb.png"
	thistype := "pool"

	d := models.Description{
		Further: further,
		Image:   image,
		Long:    long,
		Name:    &name,
		Short:   short,
		Thumb:   thumb,
		Type:    &thistype,
	}

	p := models.Pool{
		Description: &d,
		MinSession:  60,
		MaxSession:  7201,
	}
	// Now login again with booking token in body and see that subject is retained

	reqBody, err := json.Marshal(p)
	assert.NoError(t, err)
	req, err = http.NewRequest("POST", host+"/api/v1/pools", bytes.NewBuffer(reqBody))
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	pid := models.ID{}
	err = json.Unmarshal(body, &pid)
	assert.NoError(t, err)

	assert.True(t, len(*pid.ID) > 35)

	// get ID back, use ID to get description, and compare...
	req, err = http.NewRequest("GET", host+"/api/v1/pools/"+*pid.ID, nil)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	pd := models.Description{}
	err = json.Unmarshal(body, &pd)
	assert.NoError(t, err)

	assert.Equal(t, name, *pd.Name)
	assert.Equal(t, further, pd.Further)
	assert.Equal(t, image, pd.Image)
	assert.Equal(t, long, pd.Long)
	assert.Equal(t, short, pd.Short)
	assert.Equal(t, thumb, pd.Thumb)
	assert.Equal(t, thistype, *pd.Type)

}

func TestAddActivityToPoolID(t *testing.T) {
	debug := false
	// make pool, add to pool store
	// make an admin user login token, swap for booking token
	loginClaims := &lit.Token{}
	loginClaims.Audience = host
	loginClaims.Groups = []string{"everything"} //not an actual group
	loginClaims.Scopes = []string{"login:admin"}
	loginClaims.IssuedAt = ps.GetTime() - 1
	loginClaims.NotBefore = ps.GetTime() - 1
	loginClaims.ExpiresAt = loginClaims.NotBefore + ps.BookingTokenDuration
	loginToken := jwt.NewWithClaims(jwt.SigningMethodHS256, loginClaims)
	loginBearer, err := loginToken.SignedString([]byte(ps.Secret))
	assert.NoError(t, err)

	client := &http.Client{}
	req, err := http.NewRequest("POST", host+"/api/v1/login", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", loginBearer)
	resp, err := client.Do(req)
	assert.NoError(t, err)
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	btr := &models.Bookingtoken{}
	err = json.Unmarshal(body, btr)
	assert.NoError(t, err)

	if btr == nil {
		t.Fatal("no token returned")
	}
	adminBearer := *(btr.Token)

	// make a description, post in body

	further := "https://example.io/further.html"
	image := "https://example.io/image.png"
	long := "some long long long description"
	name := "Some pool name"
	short := "short story"
	thumb := "https://example.io/thumb.png"
	thistype := "pool"

	d := models.Description{
		Further: further,
		Image:   image,
		Long:    long,
		Name:    &name,
		Short:   short,
		Thumb:   thumb,
		Type:    &thistype,
	}

	p := models.Pool{
		Description: &d,
		MinSession:  60,
		MaxSession:  7201,
	}
	// Now login again with booking token in body and see that subject is retained

	reqBody, err := json.Marshal(p)
	assert.NoError(t, err)
	req, err = http.NewRequest("POST", host+"/api/v1/pools", bytes.NewBuffer(reqBody))
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	pid := models.ID{}
	err = json.Unmarshal(body, &pid)
	assert.NoError(t, err)

	assert.True(t, len(*pid.ID) > 35)
	poolID := *pid.ID

	// get ID back, use ID to get description, and compare...
	req, err = http.NewRequest("GET", host+"/api/v1/pools/"+*pid.ID, nil)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	pd := models.Description{}
	err = json.Unmarshal(body, &pd)
	assert.NoError(t, err)

	assert.Equal(t, name, *pd.Name)
	assert.Equal(t, further, pd.Further)
	assert.Equal(t, image, pd.Image)
	assert.Equal(t, long, pd.Long)
	assert.Equal(t, short, pd.Short)
	assert.Equal(t, thumb, pd.Thumb)
	assert.Equal(t, thistype, *pd.Type)

	// create activity which will pass activity check
	Further := "Further"
	ID := "ID"
	Image := "Image"
	Long := "Long"
	Short := "Short"
	Name := "Name"
	Thumb := "Thumb"
	Type := "Type"

	ad0 := models.Description{
		Further: Further,
		ID:      ID,
		Image:   Image,
		Long:    Long,
		Short:   Short,
		Name:    &Name,
		Thumb:   Thumb,
		Type:    &Type,
	}

	Audience := "https://example.com"
	ConnectionType := "session"
	Scopes := []string{"read", "write"}
	Topic := "Topic"

	ap := models.Permission{
		Audience:       &Audience,
		ConnectionType: &ConnectionType,
		Scopes:         Scopes,
		Topic:          &Topic,
	}

	For := "For"
	Token := "Token"
	URL := "URL"
	Verb := "Verb"
	Exp := float64(time.Now().Unix() + 3600)

	s0 := models.Stream{
		For:        &For,
		Permission: &ap,
		Token:      &Token,
		URL:        &URL,
		Verb:       &Verb,
	}

	s1 := s0
	roF := "roF"
	s1.For = &roF

	StreamsRequired := []string{"Streams", "Required"}

	ad1 := ad0
	ad2 := ad0

	ad1.ID = "someUI"
	ad2.ID = "anotherUI"

	URL1 := "someURL"
	URL2 := "anotherURL"

	u0 := models.UserInterface{
		Description:     &ad1,
		URL:             &URL1,
		StreamsRequired: StreamsRequired,
	}

	u1 := models.UserInterface{
		Description:     &ad2,
		URL:             &URL2,
		StreamsRequired: StreamsRequired,
	}

	ma := &models.Activity{
		Description: &ad0,
		Exp:         &Exp,
		Streams:     []*models.Stream{&s0, &s1},
		Uis:         []*models.UserInterface{&u0, &u1},
	}

	err = pool.CheckActivity(pool.NewActivityFromModel(ma))
	assert.NoError(t, err)

	if debug {
		pretty, err := json.MarshalIndent(*ma, "", "\t")
		assert.NoError(t, err)
		fmt.Println(string(pretty))
	}

	// submit to pool...

	reqBody2, err := json.Marshal(ma)
	assert.NoError(t, err)
	req, err = http.NewRequest("POST", host+"/api/v1/pools/"+poolID+"/activities", bytes.NewBuffer(reqBody2))
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	if debug {
		fmt.Println(string(body))
	}

	pid = models.ID{}
	err = json.Unmarshal(body, &pid)
	assert.NoError(t, err)

	// check not "ID" as it was originally set
	// this is POST for new, not PUT for update
	assert.True(t, len(*pid.ID) > 35)

	activityID := *pid.ID

	// Get activity description and compare
	req, err = http.NewRequest("GET", host+"/api/v1/pools/"+poolID+"/activities/"+activityID, nil)
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	if debug {
		fmt.Println(string(body))
	}
	ad := models.Description{}
	err = json.Unmarshal(body, &ad)
	assert.NoError(t, err)

	ad0.ID = activityID

	assert.Equal(t, ad0, ad)

	// Modify activity and update...
	newName := "This is the updated activity name"
	ad0.Name = &newName
	ma = &models.Activity{
		Description: &ad0,
		Exp:         &Exp,
		Streams:     []*models.Stream{&s0, &s1},
		Uis:         []*models.UserInterface{&u0, &u1},
	}
	// Swap to PUT and put new activity in body...
	reqBody3, err := json.Marshal(ma)
	assert.NoError(t, err)
	req, err = http.NewRequest("PUT", host+"/api/v1/pools/"+poolID+"/activities/"+activityID, bytes.NewBuffer(reqBody3))
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	pid = models.ID{}
	err = json.Unmarshal(body, &pid)
	assert.NoError(t, err)
	assert.Equal(t, activityID, *pid.ID)

	// Now get activity again and check name has changed
	req, err = http.NewRequest("GET", host+"/api/v1/pools/"+poolID+"/activities/"+activityID, nil)
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	if debug {
		fmt.Println(string(body))
	}
	ad = models.Description{}
	err = json.Unmarshal(body, &ad)
	assert.NoError(t, err)

	assert.Equal(t, newName, *ad.Name)
	assert.Equal(t, ad0, ad)

}

func TestTODO(t *testing.T) {

	todo := []string{
		"Delete Activity from pool",
		"Delete pool from group",
		"Remove pool from Poolstore (taking activities with it, presumably)",
		"Remove group from poolstore, but leave pools behind)",
		"Import and export current state",
		"Report current bookings to user",
		"Report max bookings limit to user",
		"Reset PoolStore to clean, known state",
		"Check if stream already exists",
		"Lock System to new bookings (set max bookings to zero)",
		"Report all current bookings and duration (plot in-use)",
	}
	for n, l := range todo {
		fmt.Println(n, ": ", l)
	}

}
