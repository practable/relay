package book

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
	"github.com/timdrysdale/relay/pkg/booking"
	"github.com/timdrysdale/relay/pkg/booking/models"
	lit "github.com/timdrysdale/relay/pkg/login"
	"github.com/timdrysdale/relay/pkg/permission"
	"github.com/timdrysdale/relay/pkg/pool"
)

var ps *pool.PoolStore
var host, secret string
var bookingDuration, mocktime, startime int64

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

	port, err := freeport.GetFreePort()
	if err != nil {
		panic(err)
	}

	host = "http://[::]:" + strconv.Itoa(port)

	go booking.API(ctx, port, host, secret, ps)

	time.Sleep(time.Second)

	exitVal := m.Run()

	os.Exit(exitVal)
}

func TestBooking(t *testing.T) {

	loginClaims := &lit.Token{}
	loginClaims.Audience = host
	loginClaims.Groups = []string{"somecourse", "everyone"}
	loginClaims.Scopes = []string{"login", "user"}
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
	assert.Equal(t, []string{"booking", "user"}, claims.Scopes)
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
	assert.Equal(t, []string{"booking", "user"}, claims.Scopes)
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
	claims.Scopes = []string{"booking", "user"}
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
	claims.Scopes = []string{"booking", "user"}
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

func TestGetPoolsByID(t *testing.T) {

	name := "stuff"

	g0 := pool.NewGroup(name)
	defer ps.DeleteGroup(g0)

	ps.AddGroup(g0)
	defer ps.DeleteGroup(g0)

	p0 := pool.NewPool("stuff0")
	p1 := pool.NewPool("stuff1")
	g0.AddPools([]*pool.Pool{p0, p1})

	claims := &lit.Token{}
	claims.Audience = host
	claims.Groups = []string{name}
	claims.Scopes = []string{"booking", "user"}
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
	q := req.URL.Query()
	q.Add("group_id", g0.ID)
	req.URL.RawQuery = q.Encode()
	assert.NoError(t, err)
	resp, err := client.Do(req)
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	r := []string{}

	err = json.Unmarshal(body, &r)
	assert.NoError(t, err)
	assert.Equal(t, []string{p0.ID, p1.ID}, r)

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
	loginClaims.Scopes = []string{"login", "user"}
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
	req, err = http.NewRequest("GET", host+"/api/v1/pools/"+p0.ID+"/description", nil)
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
	loginClaims.Scopes = []string{"login", "user"}
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
	assert.Equal(t, pt0.Topic, stc.Topic)
	assert.Equal(t, ps.Now()+2000, stc.ExpiresAt)

	assert.Equal(t, "https://example.com/session/123data", *(ma.Streams[0].URL))
	assert.Equal(t, "https://example.com/session/456video", *(ma.Streams[1].URL))
	assert.Equal(t, "https://static.example.com/example.html?data={{data}}\u0026video={{video}}", *(ma.Uis[0].URL))
	assert.Equal(t, []string{"data", "video"}, ma.Uis[0].StreamsRequired)

}
