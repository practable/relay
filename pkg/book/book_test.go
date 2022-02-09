package book

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
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
	"github.com/xtgo/uuid"
)

var debug bool
var l *bookingstore.Limit
var ps *pool.PoolStore
var host, secret string
var bookingDuration, mocktime, startime int64

// Deferred deletes are to clean up between tests
// and are not an example of how to use the system
// in production - you want the items to live on
// so that some booking can be done!

// If you get unexplained issues with tests, then disable
// TestUnmarshalMarshalPoolStore
// because your new mods may have broken import and export
// which is tested on the common poolstore that all tests
// in this file rely on. Obvs fix import and export if needed
// and re-emable this test after triaging your other issues.

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

func TestGetSetLockedMessage(t *testing.T) {
	loginClaims := &lit.Token{}
	loginClaims.Audience = host
	loginClaims.Groups = []string{"stuff", "everyone"}
	loginClaims.Scopes = []string{"login:admin"}
	loginClaims.IssuedAt = ps.GetTime() - 1
	loginClaims.NotBefore = ps.GetTime() - 1
	loginClaims.ExpiresAt = loginClaims.NotBefore + ps.BookingTokenDuration
	loginToken := jwt.NewWithClaims(jwt.SigningMethodHS256, loginClaims)
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

	adminBearer := *(btr.Token)

	/* keep original locked status and  message for putting back later
	      (this test doesn't really
		  care what the default is, but other tests might) */

	req, err = http.NewRequest("GET", host+"/api/v1/admin/status", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	ms := &models.StoreStatus{}
	err = json.Unmarshal(body, ms)
	assert.NoError(t, err)

	originalLock := ms.Locked
	originalMessage := ms.Msg

	req, err = http.NewRequest("POST", host+"/api/v1/admin/status", nil)
	assert.NoError(t, err)
	q := req.URL.Query()
	q.Add("lock", "true")
	q.Add("msg", "A different message")
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	ms = &models.StoreStatus{}
	err = json.Unmarshal(body, ms)
	assert.NoError(t, err)

	assert.Equal(t, true, ms.Locked)
	assert.Equal(t, "A different message", ms.Msg)

	req, err = http.NewRequest("POST", host+"/api/v1/admin/status", nil)
	assert.NoError(t, err)
	q = req.URL.Query()
	if originalLock {
		q.Add("lock", "true")
	} else {
		q.Add("lock", "false")
	}
	q.Add("msg", originalMessage)
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	ms = &models.StoreStatus{}
	err = json.Unmarshal(body, ms)
	assert.NoError(t, err)

	assert.Equal(t, originalLock, ms.Locked)
	assert.Equal(t, originalMessage, ms.Msg)

}

func TestBooking(t *testing.T) {

	// need a pool to check we don't duplicate pools
	g0 := pool.NewGroup("everyone")
	ps.AddGroup(g0)
	defer ps.DeleteGroup(g0)

	p0 := pool.NewPool("stuff0")
	ps.AddPool(p0)
	defer ps.DeletePool(p0)

	g0.AddPool(p0)

	g1 := pool.NewGroup("somecourse")
	ps.AddGroup(g1)
	defer ps.DeleteGroup(g1)
	g1.AddPool(p0) //add to both groups - should only see it once though

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

	// Now login again with previous booking token in body and see that subject is retained
	// but that only groups in new token are returned
	newLoginClaims := &lit.Token{}
	newLoginClaims.Audience = host
	newLoginClaims.Groups = []string{"othercourse", "everyone"}
	newLoginClaims.Scopes = []string{"login:user"}
	newLoginClaims.IssuedAt = ps.GetTime() - 1
	newLoginClaims.NotBefore = ps.GetTime() - 1
	newLoginClaims.ExpiresAt = newLoginClaims.NotBefore + ps.BookingTokenDuration
	// sign user token
	newLoginToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newLoginClaims)
	// Sign and get the complete encoded token as a string using the secret
	newLoginBearer, err := newLoginToken.SignedString([]byte(ps.Secret))
	assert.NoError(t, err)

	respBody, err := json.Marshal(lit.TokenInBody{Token: bookingTokenReturned})
	assert.NoError(t, err)
	req, err = http.NewRequest("POST", host+"/api/v1/login", bytes.NewBuffer(respBody))
	assert.NoError(t, err)
	req.Header.Set("Content-type", "application/json")
	req.Header.Add("Authorization", newLoginBearer)
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

	//note groups are different  somecourse -> othercourse
	assert.Equal(t, []string{"othercourse", "everyone"}, claims.Groups)
	assert.Equal(t, []string{"booking:user"}, claims.Scopes)
	assert.True(t, claims.ExpiresAt < ps.Now()+bookingDuration+15)
	assert.True(t, claims.ExpiresAt > ps.Now()+bookingDuration-15)
	assert.True(t, len(claims.Subject) >= 35)

	// key test
	assert.Equal(t, subject, claims.Subject)
	assert.Equal(t, 1, len(claims.Pools))

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

	assert.True(t, util.SortCompare([]string{p0.ID, p1.ID}, claims.Pools))

	// Check we can get our booking info, even with no bookings made
	req, err = http.NewRequest("GET", host+"/api/v1/login", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bookingBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	bk := models.Bookings{}
	err = json.Unmarshal(body, &bk)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), *bk.Max)
	assert.Equal(t, []*models.Activity{}, bk.Activities)
	assert.Equal(t, false, bk.Locked)
	assert.Equal(t, "Open for bookings", bk.Msg)

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
	assert.Equal(t, int64(2), *s.Available)
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
	assert.Equal(t, int64(0), *s.Available)
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
	assert.Equal(t, int64(0), *s.Available)
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
	assert.Equal(t, int64(0), *s.Available)
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

	streamTokenString0 := (ma.Streams[0]).Token
	//streamTokenString1 := (ma.Streams[1]).Token

	assert.Equal(t, "a", *(ma.Description.Name))
	assert.Equal(t, 2, len(ma.Streams))
	assert.Equal(t, 2, len(ma.Uis))
	//assert.Equal(t, "ey", ((ma.Streams[0]).Token))[0:2]
	//assert.Equal(t, "ey", streamTokenString1[0:2])

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

	// Check we can get our booked activity via booking info
	// Check we can get our booking info, even with no bookings made
	req, err = http.NewRequest("GET", host+"/api/v1/login", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bookingBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	bk := models.Bookings{}
	err = json.Unmarshal(body, &bk)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), *bk.Max)
	assert.Equal(t, []*models.Activity{ma}, bk.Activities)
	assert.Equal(t, false, bk.Locked)
	assert.Equal(t, "Open for bookings", bk.Msg)

}

//*******************************************************
//  _____ ___ ___ _____   _    ___ __  __ ___ _____ ___
// |_   _| __/ __|_   _| | |  |_ _|  \/  |_ _|_   _/ __|
//   | | | _|\__ \ | |   | |__ | || |\/| || |  | | \__ \
//   |_| |___|___/ |_|   |____|___|_|  |_|___| |_| |___/
//
//*******************************************************

func TestLimits(t *testing.T) {
	// minimal activity just for testing- less complete than you'd need in production
	// note that stream order and activity order are not guaranteed - hence the
	// conveniences taken in this test (which is checking limits, not token formation)

	statusCodes := []int{}
	expectedCodes := []int{}

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

	//            _           _         _             _
	//   __ _  __| |_ __ ___ (_)_ __   | | ___   __ _(_)_ __
	//  / _` |/ _` | '_ ` _ \| | '_ \  | |/ _ \ / _` | | '_ \
	// | (_| | (_| | | | | | | | | | | | | (_) | (_| | | | | |
	//  \__,_|\__,_|_| |_| |_|_|_| |_| |_|\___/ \__, |_|_| |_|
	//                                          |___/

	loginClaims := &lit.Token{}
	loginClaims.Audience = host
	loginClaims.Groups = []string{name, "everyone"}
	loginClaims.Scopes = []string{"login:admin"}
	loginClaims.IssuedAt = ps.GetTime() - 1
	loginClaims.NotBefore = ps.GetTime() - 1
	loginClaims.ExpiresAt = loginClaims.NotBefore + ps.BookingTokenDuration
	loginToken := jwt.NewWithClaims(jwt.SigningMethodHS256, loginClaims)
	loginBearer, err := loginToken.SignedString([]byte(ps.Secret))
	assert.NoError(t, err)

	mocktime = time.Now().Unix()

	client := &http.Client{}
	req, err := http.NewRequest("POST", host+"/api/v1/login", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", loginBearer)
	resp, err := client.Do(req)
	assert.NoError(t, err)
	statusCodes = append(statusCodes, resp.StatusCode)
	expectedCodes = append(expectedCodes, http.StatusOK)

	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	btr := &models.Bookingtoken{}
	err = json.Unmarshal(body, btr)
	assert.NoError(t, err)

	if btr == nil {
		t.Fatal("no token returned")
	}

	adminBearer := *(btr.Token)
	//  _         _
	// | |___  __| |__
	// | / _ \/ _| / /
	// |_\___/\__|_\_\
	//
	// lock

	req, err = http.NewRequest("POST", host+"/api/v1/admin/status", nil)
	assert.NoError(t, err)
	q := req.URL.Query()
	q.Add("lock", "true")
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	statusCodes = append(statusCodes, resp.StatusCode)
	expectedCodes = append(expectedCodes, http.StatusOK)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	ms := &models.StoreStatus{}
	err = json.Unmarshal(body, ms)
	assert.NoError(t, err)

	assert.Equal(t, true, ms.Locked)

	//                     _           _
	//  _  _ ___ ___ _ _  | |___  __ _(_)_ _
	// | || (_-</ -_) '_| | / _ \/ _` | | ' \
	//  \_,_/__/\___|_|   |_\___/\__, |_|_||_|
	//                           |___/
	//
	// user login

	loginClaims = &lit.Token{}
	loginClaims.Audience = host
	//check that missing group "everyone" in PoolStore does not stop login
	loginClaims.Groups = []string{name, "everyone"}
	loginClaims.Scopes = []string{"login:user"}
	loginClaims.IssuedAt = ps.GetTime() - 1
	loginClaims.NotBefore = ps.GetTime() - 1
	loginClaims.ExpiresAt = loginClaims.NotBefore + ps.BookingTokenDuration
	// sign user token
	loginToken = jwt.NewWithClaims(jwt.SigningMethodHS256, loginClaims)
	// Sign and get the complete encoded token as a string using the secret
	loginBearer, err = loginToken.SignedString([]byte(ps.Secret))
	assert.NoError(t, err)

	mocktime = time.Now().Unix()

	client = &http.Client{}
	req, err = http.NewRequest("POST", host+"/api/v1/login", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", loginBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	statusCodes = append(statusCodes, resp.StatusCode)
	expectedCodes = append(expectedCodes, http.StatusOK)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	btr = &models.Bookingtoken{}
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

	//  _              __        __      _ _
	// | |_ _ _ _  _  / _|___   / _|__ _(_) |
	// |  _| '_| || | > _|_ _| |  _/ _` | | |
	//  \__|_|  \_, | \_____|  |_| \__,_|_|_|
	//          |__/
	//
	// try & fail

	req, err = http.NewRequest("POST", host+"/api/v1/pools/"+p0.ID+"/sessions", nil)
	assert.NoError(t, err)
	q = req.URL.Query()
	q.Add("duration", "2000")
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Authorization", bookingBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	statusCodes = append(statusCodes, resp.StatusCode)
	expectedCodes = append(expectedCodes, http.StatusPaymentRequired)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	assert.Equal(t, "\"No new sessions allowed. Try again later.\"\n", string(body))

	assert.Equal(t, http.StatusPaymentRequired, resp.StatusCode)

	//            _         _
	//  _  _ _ _ | |___  __| |__
	// | || | ' \| / _ \/ _| / /
	//  \_,_|_||_|_\___/\__|_\_\
	//
	// unlock

	req, err = http.NewRequest("POST", host+"/api/v1/admin/status", nil)
	assert.NoError(t, err)
	q = req.URL.Query()
	q.Add("lock", "false")
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	statusCodes = append(statusCodes, resp.StatusCode)
	expectedCodes = append(expectedCodes, http.StatusOK)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	ms = &models.StoreStatus{}
	err = json.Unmarshal(body, ms)
	assert.NoError(t, err)

	assert.Equal(t, false, ms.Locked)

	//                           _             _   _     _ _
	//  _ _ ___ __ _ _  _ ___ __| |_   __ _ __| |_(_)_ _(_) |_ _  _
	// | '_/ -_) _` | || / -_|_-<  _| / _` / _|  _| \ V / |  _| || |
	// |_| \___\__, |\_,_\___/__/\__| \__,_\__|\__|_|\_/|_|\__|\_, |
	//            |_|                                          |__/
	//
	// request an activity...
	req, err = http.NewRequest("POST", host+"/api/v1/pools/"+p0.ID+"/sessions", nil)
	assert.NoError(t, err)
	q = req.URL.Query()
	q.Add("duration", "2000")
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Authorization", bookingBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	statusCodes = append(statusCodes, resp.StatusCode)
	expectedCodes = append(expectedCodes, http.StatusOK)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	ma := &models.Activity{}
	err = json.Unmarshal(body, ma)
	assert.NoError(t, err)

	if ma == nil {
		t.Fatal("no token returned")
	}

	streamTokenString0 := (ma.Streams[0]).Token

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
	expectedCodes = append(expectedCodes, http.StatusOK)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	ma = &models.Activity{}
	err = json.Unmarshal(body, ma)
	assert.NoError(t, err)

	if ma == nil {
		t.Fatal("no token returned")
	}

	streamTokenString0 = (ma.Streams[0]).Token

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

	// Now let's try being a different user -
	req, err = http.NewRequest("POST", host+"/api/v1/login", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", loginBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	statusCodes = append(statusCodes, resp.StatusCode)
	expectedCodes = append(expectedCodes, http.StatusOK)

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
	// we should get a 404 not found (no kit left)
	// but no payment required, because we're a different user
	// with no bookings at present, so we are under quota
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
	expectedCodes = append(expectedCodes, http.StatusNotFound)

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
	assert.Equal(t, "\"Maximum concurrent sessions already reached. Try again later.\"\n", string(body))
	statusCodes = append(statusCodes, resp.StatusCode)
	expectedCodes = append(expectedCodes, http.StatusPaymentRequired)
	assert.Equal(t, expectedCodes, statusCodes)
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
		Token:      Token,
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

	// add config URL - in this check we only find out if it causes an error
	// not if it is properly recorded - so we should check it elsewhere
	configURL := "https://somewhere.com/config/config.json"

	cfg := models.Config{
		URL: &configURL,
	}

	ma := &models.Activity{
		Config:      &cfg,
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

	log.Debugf("Description returned: %s", string(body))

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

//***********************************************************
//                               _         _
//  _  _ _ _  _ __  __ _ _ _ __| |_  __ _| |
// | || | ' \| '  \/ _` | '_(_-< ' \/ _` | |
//  \_,_|_||_|_|_|_\__,_|_| /__/_||_\__,_|_|
//
//***********************************************************
func TestUnmarshalMarshalPoolStore(t *testing.T) {

	// Set up a pool, import, export, then run same test as above
	// test: TestLimits

	// *** Setup groups, pools, activities *** //

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

	// *** Unmarshal and marshal the pools *** //

	p, err := json.Marshal(ps)
	assert.NoError(t, err)

	ps2 := &pool.PoolStore{}

	err = json.Unmarshal(p, &ps2)
	assert.NoError(t, err)

	p2, err := json.MarshalIndent(ps2, "", "\t")
	assert.NoError(t, err)

	if debug {
		fmt.Println(string(p2))
	}

	// *** Initialise the imported poolstore, then swap it for the live one *** //
	ps2.PostImportEssential()

	// we're just using mocktime to keep up with real time, so this isn't really needed
	// you can comment it out and this test still passes
	ps2.PostImportSetNow(func() int64 { return func(now *int64) int64 { return *now }(&mocktime) })

	// This will ruin other tests unless it works ok ....
	ps = ps2

	// *** run the rest of the test *** //

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

	streamTokenString0 := (ma.Streams[0]).Token

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

	streamTokenString0 = (ma.Streams[0]).Token

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
	assert.Equal(t, "\"Maximum concurrent sessions already reached. Try again later.\"\n", string(body))
	statusCodes = append(statusCodes, resp.StatusCode)
	assert.Equal(t, []int{200, 200, 404, 402}, statusCodes)

}

//***********************************************************
//
//  _                     _                           _
// (_)_ __  _ __  ___ _ _| |_   _____ ___ __  ___ _ _| |_
// | | '  \| '_ \/ _ \ '_|  _| / -_) \ / '_ \/ _ \ '_|  _|
// |_|_|_|_| .__/\___/_|  \__| \___/_\_\ .__/\___/_|  \__|
//         |_|                         |_|
//
//
//   ___ ___ _____   ___  ___   ___  _  _____ _  _  ___ ___
//  / __| __|_   _| | _ )/ _ \ / _ \| |/ /_ _| \| |/ __/ __|
// | (_ | _|  | |   | _ \ (_) | (_) | ' < | || .` | (_ \__ \
//  \___|___| |_|   |___/\___/ \___/|_|\_\___|_|\_|\___|___/
//
//
//***********************************************************
// http://patorjk.com/software/taag/#p=display&f=Small&t=import%20export

func TestImportExportPoolStoreGetCurrentBookings(t *testing.T) {

	veryVerbose := false
	// Set up a Local pool, import via server, interact, export, check for bookings
	// *** Setup groups, pools, activities *** //

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ps2 := pool.NewPoolStore().
		WithSecret(secret).
		WithBookingTokenDuration(bookingDuration).
		WithNow(func() int64 { return func(now *int64) int64 { return *now }(&mocktime) })

	l2 := bookingstore.New(ctx).WithFlush(time.Minute).WithMax(2).WithProvisionalPeriod(5 * time.Second)

	statusCodes := []int{}

	name := "stuff"
	g0 := pool.NewGroup(name)
	defer ps2.DeleteGroup(g0)

	ps2.AddGroup(g0)
	defer ps2.DeleteGroup(g0)

	p0 := pool.NewPool("stuff0").WithNow(func() int64 { return func(now *int64) int64 { return *now }(&mocktime) })

	g0.AddPool(p0)
	ps2.AddPool(p0)
	defer ps2.DeletePool(p0)

	a := pool.NewActivity("a", ps2.Now()+3600)

	cu0 := "https://somewhere.com/config/config0.json"

	a.Config = pool.Config{URL: cu0}

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

	a2 := pool.NewActivity("a2", ps2.Now()+3600)

	cu1 := "https://somewhere.com/config/config1.json"
	a2.Config = pool.Config{URL: cu1}

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

	// *** Unmarshal and marshal the pools *** //

	bps2, err := ps2.ExportAll()
	assert.NoError(t, err)
	bl2, err := l2.ExportAll()
	assert.NoError(t, err)
	bookingEnc := base64.StdEncoding.EncodeToString(bl2)
	poolEnc := base64.StdEncoding.EncodeToString(bps2)

	store := models.Poolstore{
		Booking: &bookingEnc,
		Pool:    &poolEnc,
	}
	var pretty []byte

	if veryVerbose {
		pretty, err = json.MarshalIndent(ps2, "", "\t")
		fmt.Println("LOCAL POOL STORE")
		fmt.Println(string(pretty))
		fmt.Println("LOCAL POOL STORE marshalled to bytes")
		fmt.Println(string(bps2))
		fmt.Println("LOCAL POOL STORE Base64 Encoded")
		fmt.Println(string(poolEnc))
		fmt.Println("LOCAL POOL STORE Base64 Decoded")
		poolDec, err := base64.StdEncoding.DecodeString(poolEnc)
		assert.NoError(t, err)
		fmt.Println(string(poolDec))
		pscheck := &pool.PoolStore{}
		err = json.Unmarshal(poolDec, pscheck)
		assert.NoError(t, err)
		pretty, err = json.MarshalIndent(pscheck, "", "\t")
		fmt.Println("LOCAL POOL STORE Unmarshalled")
		fmt.Println(string(pretty))
	}
	mocktime = time.Now().Unix()

	//            _           _         _             _
	//   __ _  __| |_ __ ___ (_)_ __   | | ___   __ _(_)_ __
	//  / _` |/ _` | '_ ` _ \| | '_ \  | |/ _ \ / _` | | '_ \
	// | (_| | (_| | | | | | | | | | | | | (_) | (_| | | | | |
	//  \__,_|\__,_|_| |_| |_|_|_| |_| |_|\___/ \__, |_|_| |_|
	//                                          |___/

	loginClaims := &lit.Token{}
	loginClaims.Audience = host
	loginClaims.Groups = []string{name, "everyone"}
	loginClaims.Scopes = []string{"login:admin"}
	loginClaims.IssuedAt = ps.GetTime() - 1
	loginClaims.NotBefore = ps.GetTime() - 1
	loginClaims.ExpiresAt = loginClaims.NotBefore + ps.BookingTokenDuration
	loginToken := jwt.NewWithClaims(jwt.SigningMethodHS256, loginClaims)
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

	adminBearer := *(btr.Token)

	token, err := jwt.ParseWithClaims(adminBearer, &lit.Token{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)

	claims, ok := token.Claims.(*lit.Token)

	if veryVerbose {
		pretty, err := json.MarshalIndent(claims, "", "\t")
		assert.NoError(t, err)
		fmt.Println("------------- adminBearer claims-----------")
		fmt.Println(string(pretty))
	}

	assert.True(t, ok)
	assert.True(t, token.Valid)

	// adminBearer doesn't get pools evaluated because it can access them all

	//  _                            _
	// (_)_ __ ___  _ __   ___  _ __| |_
	// | | '_ ` _ \| '_ \ / _ \| '__| __|
	// | | | | | | | |_) | (_) | |  | |_
	// |_|_| |_| |_| .__/ \___/|_|   \__|
	//             |_|

	reqBody, err := json.Marshal(store)

	req, err = http.NewRequest("POST", host+"/api/v1/admin/poolstore", bytes.NewBuffer(reqBody))
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	req.Header.Add("Content-type", "application/json")
	resp, err = client.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	if veryVerbose {
		t.Log("importStatus:", resp.Status)
	}

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	ms := &models.StoreStatus{}
	err = json.Unmarshal(body, ms)
	assert.NoError(t, err)

	if ms == nil {
		t.Fatal("no status returned")
	}

	if veryVerbose {
		pretty, err := json.MarshalIndent(ms, "", "\t")
		assert.NoError(t, err)
		fmt.Println("-------------STORE STATUS AFTER IMPORT-----------")
		fmt.Println(string(pretty))
	}

	//                      _             _
	//  _   _ ___  ___ _ __| | ___   __ _(_)_ __
	// | | | / __|/ _ \ '__| |/ _ \ / _` | | '_ \
	// | |_| \__ \  __/ |  | | (_) | (_| | | | | |
	//  \__,_|___/\___|_|  |_|\___/ \__, |_|_| |_|
	//                              |___/

	// login
	loginClaims = &lit.Token{}
	loginClaims.Audience = host
	//check that missing group "everyone" in PoolStore does not stop login
	loginClaims.Groups = []string{name, "everyone"}
	loginClaims.Scopes = []string{"login:user"}
	loginClaims.IssuedAt = ps.GetTime() - 1
	loginClaims.NotBefore = ps.GetTime() - 1
	loginClaims.ExpiresAt = loginClaims.NotBefore + ps.BookingTokenDuration
	// sign user token
	loginToken = jwt.NewWithClaims(jwt.SigningMethodHS256, loginClaims)
	// Sign and get the complete encoded token as a string using the secret
	loginBearer, err = loginToken.SignedString([]byte(ps.Secret))
	assert.NoError(t, err)

	mocktime = time.Now().Unix()

	client = &http.Client{}
	req, err = http.NewRequest("POST", host+"/api/v1/login", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", loginBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	btr = &models.Bookingtoken{}
	err = json.Unmarshal(body, btr)
	assert.NoError(t, err)

	if btr == nil {
		t.Fatal("no token returned")
	}

	bookingBearer := *(btr.Token)

	token, err = jwt.ParseWithClaims(bookingBearer, &lit.Token{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)

	claims, ok = token.Claims.(*lit.Token)

	if veryVerbose {
		pretty, err = json.MarshalIndent(claims, "", "\t")
		fmt.Println("Login token claims")
		fmt.Println(string(pretty))
	}
	assert.True(t, ok)
	assert.True(t, token.Valid)

	assert.Equal(t, []string{p0.ID}, claims.Pools)

	//                                 _                _   _       _ _
	//  _ __ ___  __ _ _   _  ___  ___| |_    __ _  ___| |_(_)_   _(_) |_ _   _
	// | '__/ _ \/ _` | | | |/ _ \/ __| __|  / _` |/ __| __| \ \ / / | __| | | |
	// | | |  __/ (_| | |_| |  __/\__ \ |_  | (_| | (__| |_| |\ V /| | |_| |_| |
	// |_|  \___|\__, |\__,_|\___||___/\__|  \__,_|\___|\__|_| \_/ |_|\__|\__, |
	//              |_|                                                   |___/
	//

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

	assert.True(t, ma.Config.URL != nil)
	assert.True(t, ma.Description.Name != nil)

	assert.True(t, (*ma.Config.URL == cu0 && *ma.Description.Name == "a") || (*ma.Config.URL == cu1 && *ma.Description.Name == "a2"))

	log.Debugf("Config:%s", *ma.Config.URL)
	log.Debugf("Name:%s\n", *ma.Description.Name)

	streamTokenString0 := (ma.Streams[0]).Token

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

	streamTokenString0 = (ma.Streams[0]).Token

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
	assert.Equal(t, "\"Maximum concurrent sessions already reached. Try again later.\"\n", string(body))
	statusCodes = append(statusCodes, resp.StatusCode)
	assert.Equal(t, []int{200, 200, 404, 402}, statusCodes)

	/* Get the StoreStatus */
	//     _                    _        _
	//  __| |_ ___ _ _ ___   __| |_ __ _| |_ _  _ ___
	// (_-<  _/ _ \ '_/ -_) (_-<  _/ _` |  _| || (_-<
	// /__/\__\___/_| \___| /__/\__\__,_|\__|\_,_/__/
	//
	req, err = http.NewRequest("GET", host+"/api/v1/admin/status", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	ms = &models.StoreStatus{}
	err = json.Unmarshal(body, ms)
	assert.NoError(t, err)

	if veryVerbose {
		pretty, err = json.MarshalIndent(ms, "", "\t")
		fmt.Println(string(pretty))
	}

	assert.Equal(t, int64(2), ms.Activities)
	assert.Equal(t, int64(2), ms.Bookings)
	assert.Equal(t, int64(1), ms.Groups)
	assert.Equal(t, int64(1), ms.Pools)
	assert.Equal(t, float64(mocktime+2000), ms.LastBookingEnds)

	// 	   ___ ___ _____   ___  ___   ___  _  _____ _  _  ___ ___
	//   / __| __|_   _| | _ )/ _ \ / _ \| |/ /_ _| \| |/ __/ __|
	//  | (_ | _|  | |   | _ \ (_) | (_) | ' < | || .` | (_ \__ \
	//   \___|___| |_|   |___/\___/ \___/|_|\_\___|_|\_|\___|___/
	//
	req, err = http.NewRequest("GET", host+"/api/v1/login", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", bookingBearer) //different user this time
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	mb := &models.Bookings{}

	err = json.Unmarshal(body, mb)
	assert.NoError(t, err)

	assert.Equal(t, int64(2), *mb.Max)
	assert.Equal(t, 2, len(mb.Activities))
	assert.False(t, mb.Locked)
	assert.Equal(t, "Open for bookings", mb.Msg)

	// Check that there is a proper permissions token in a stream
	// While this is a necessary rather than a sufficient check
	// it's a good start. TODO - make this check more complete in future
	// so that we don't forget what we need in this and end up
	// breaking the booking client (webapp)

	act0 := mb.Activities[0]
	stream0 := act0.Streams[0]

	ptclaims = &permission.Token{}

	streamToken, err = jwt.ParseWithClaims(stream0.Token, ptclaims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method was %v", token.Header["alg"])
		}
		return []byte(ps.Secret), nil
	})

	if err != nil {
		t.Error(err)
	}

	stc, ok = streamToken.Claims.(*permission.Token)

	assert.True(t, ok)

	assert.True(t, stc.Topic == "foo" || stc.Topic == "bar")

	if veryVerbose {
		pretty, err = json.MarshalIndent(act0, "", "\t")
		fmt.Println(string(pretty))
	}

	//   _____ ___ __  ___ _ _| |_
	//  / -_) \ / '_ \/ _ \ '_|  _|
	//  \___/_\_\ .__/\___/_|  \__|
	//          |_|
	//
	// EXPORT AGAIN AND CHECK?

	req, err = http.NewRequest("GET", host+"/api/v1/admin/poolstore", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	export := &models.Poolstore{}
	err = json.Unmarshal(body, export)
	assert.NoError(t, err)

	poolBytes, err := base64.StdEncoding.DecodeString(*export.Pool)
	assert.NoError(t, err)

	bookingBytes, err := base64.StdEncoding.DecodeString(*export.Booking)
	assert.NoError(t, err)

	exportedPool := &pool.PoolStore{}

	err = json.Unmarshal(poolBytes, exportedPool)
	assert.NoError(t, err)

	exportedBooking := &bookingstore.Limit{}

	err = json.Unmarshal(bookingBytes, exportedBooking)
	assert.NoError(t, err)

	if veryVerbose {
		prettyPool, err := json.MarshalIndent(exportedPool, "", "\t")
		assert.NoError(t, err)
		fmt.Println(string(prettyPool))

		prettyBooking, err := json.MarshalIndent(exportedBooking, "", "\t")
		assert.NoError(t, err)
		fmt.Println(string(prettyBooking))
	}

	assert.Equal(t, 2, len(exportedPool.Pools[p0.ID].Activities))
	assert.Equal(t, 2, len(exportedBooking.ActivityBySession))
	assert.Equal(t, 2, len(exportedBooking.UserBySession))

}

//********************************************************
//    _   ___  ___    ___  ___ _    ___ _____ ___
//   /_\ |   \|   \  |   \| __| |  | __|_   _| __|
//  / _ \| |) | |) | | |) | _|| |__| _|  | | | _|
// /_/ \_\___/|___/  |___/|___|____|___| |_| |___|
//
//
//   ___ ___  ___  _   _ ___   ___  ___   ___  _
//  / __| _ \/ _ \| | | | _ \ | _ \/ _ \ / _ \| |
// | (_ |   / (_) | |_| |  _/ |  _/ (_) | (_) | |__
//  \___|_|_\\___/_\___/|_|  _|_|_ \___/_\___/|____|
//    /_\ / __|_   _|_ _\ \ / /_ _|_   _\ \ / /
//   / _ \ (__  | |  | | \ V / | |  | |  \ V /
//  /_/ \_\___| |_| |___| \_/ |___| |_|   |_|
//
//******************************************************
func TestAddDeleteGroupPoolActivity(t *testing.T) {

	veryVerbose := false
	var pretty []byte

	// Admin login

	loginClaims := &lit.Token{}
	loginClaims.Audience = host
	loginClaims.Groups = []string{"everyone"}
	loginClaims.Scopes = []string{"login:admin"}
	loginClaims.IssuedAt = ps.GetTime() - 1
	loginClaims.NotBefore = ps.GetTime() - 1
	loginClaims.ExpiresAt = loginClaims.NotBefore + ps.BookingTokenDuration
	loginToken := jwt.NewWithClaims(jwt.SigningMethodHS256, loginClaims)
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

	adminBearer := *(btr.Token)

	token, err := jwt.ParseWithClaims(adminBearer, &lit.Token{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)

	_, ok := token.Claims.(*lit.Token)
	assert.True(t, ok)
	assert.True(t, token.Valid)

	// Import an empty store (reset!!)

	req, err = http.NewRequest("DELETE", host+"/api/v1/admin/poolstore", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	if veryVerbose {
		t.Log("importStatus:", resp.Status)
	}

	/* Get the StoreStatus */
	//     _                    _        _
	//  __| |_ ___ _ _ ___   __| |_ __ _| |_ _  _ ___
	// (_-<  _/ _ \ '_/ -_) (_-<  _/ _` |  _| || (_-<
	// /__/\__\___/_| \___| /__/\__\__,_|\__|\_,_/__/
	//
	req, err = http.NewRequest("GET", host+"/api/v1/admin/status", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	ms := &models.StoreStatus{}
	err = json.Unmarshal(body, ms)
	assert.NoError(t, err)

	if veryVerbose {
		pretty, err = json.MarshalIndent(ms, "", "\t")
		fmt.Println(string(pretty))
	}

	assert.Equal(t, int64(0), ms.Activities)
	assert.Equal(t, int64(0), ms.Bookings)
	assert.Equal(t, int64(0), ms.Groups)
	assert.Equal(t, int64(0), ms.Pools)

	//          _    _                  _
	//  __ _ __| |__| |  _ __  ___  ___| |___
	// / _` / _` / _` | | '_ \/ _ \/ _ \ (_-<
	// \__,_\__,_\__,_| | .__/\___/\___/_/__/
	//                  |_|
	//

	// make a description, post in body

	further := "https://example.io/further.html"
	image := "https://example.io/image.png"
	long := "some long long long description"
	name := "red"
	short := "short story"
	thumb := "https://example.io/thumb.png"
	thistype := "pool"

	d0 := models.Description{
		Further: further,
		Image:   image,
		Long:    long,
		Name:    &name,
		Short:   short,
		Thumb:   thumb,
		Type:    &thistype,
	}

	p0 := models.Pool{
		Description: &d0,
		MinSession:  60,
		MaxSession:  7201,
	}

	reqBody, err := json.Marshal(p0)
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

	pid0 := *pid.ID
	_, err = uuid.Parse(pid0)
	assert.NoError(t, err)

	further1 := "https://example.io/further1.html"
	image1 := "https://example.io/image1.png"
	long1 := "some long long long description1"
	name1 := "red1"
	short1 := "short story1"
	thumb1 := "https://example.io/thumb1.png"
	thistype1 := "pool"

	d1 := models.Description{
		Further: further1,
		Image:   image1,
		Long:    long1,
		Name:    &name1,
		Short:   short1,
		Thumb:   thumb1,
		Type:    &thistype1,
	}

	p1 := models.Pool{
		Description: &d1,
		MinSession:  60,
		MaxSession:  7201,
	}

	reqBody, err = json.Marshal(p1)
	assert.NoError(t, err)
	req, err = http.NewRequest("POST", host+"/api/v1/pools", bytes.NewBuffer(reqBody))
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	pid = models.ID{}
	err = json.Unmarshal(body, &pid)
	assert.NoError(t, err)

	pid1 := *pid.ID
	_, err = uuid.Parse(pid1)
	assert.NoError(t, err)

	// 	          _    _
	//   __ _ __| |__| |  __ _ _ _ ___ _  _ _ __
	//  / _` / _` / _` | / _` | '_/ _ \ || | '_ \
	//  \__,_\__,_\__,_| \__, |_| \___/\_,_| .__/
	//                   |___/             |_|
	//

	further2 := "https://example.io/further2.html"
	image2 := "https://example.io/image2.png"
	long2 := "some long long long description2"
	name2 := "red2"
	short2 := "short story2"
	thumb2 := "https://example.io/thumb2.png"
	thistype2 := "group"

	d2 := models.Description{
		Further: further2,
		Image:   image2,
		Long:    long2,
		Name:    &name2,
		Short:   short2,
		Thumb:   thumb2,
		Type:    &thistype2,
	}

	pools := []string{pid0, pid1}

	mg := models.Group{
		Description: &d2,
		Pools:       pools,
	}

	reqBody, err = json.Marshal(mg)
	assert.NoError(t, err)
	req, err = http.NewRequest("POST", host+"/api/v1/groups", bytes.NewBuffer(reqBody))
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		fmt.Println("1", string(body))
	}

	pid = models.ID{}
	err = json.Unmarshal(body, &pid)
	assert.NoError(t, err)

	gid := *pid.ID
	_, err = uuid.Parse(gid)
	assert.NoError(t, err)

	/* Get the StoreStatus again */
	//     _                    _        _
	//  __| |_ ___ _ _ ___   __| |_ __ _| |_ _  _ ___
	// (_-<  _/ _ \ '_/ -_) (_-<  _/ _` |  _| || (_-<
	// /__/\__\___/_| \___| /__/\__\__,_|\__|\_,_/__/
	//
	req, err = http.NewRequest("GET", host+"/api/v1/admin/status", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	ms = &models.StoreStatus{}
	err = json.Unmarshal(body, ms)
	assert.NoError(t, err)

	if veryVerbose {
		pretty, err = json.MarshalIndent(ms, "", "\t")
		fmt.Println(string(pretty))
	}

	assert.Equal(t, int64(0), ms.Activities)
	assert.Equal(t, int64(0), ms.Bookings)
	assert.Equal(t, int64(1), ms.Groups)
	assert.Equal(t, int64(2), ms.Pools)

	// Third Pool...

	further3 := "https://example.io/further3.html"
	image3 := "https://example.io/image3.png"
	long3 := "some long long long description3"
	name3 := "red3"
	short3 := "short story3"
	thumb3 := "https://example.io/thumb3.png"
	thistype3 := "pool"

	d3 := models.Description{
		Further: further3,
		Image:   image3,
		Long:    long3,
		Name:    &name3,
		Short:   short3,
		Thumb:   thumb3,
		Type:    &thistype3,
	}

	p3 := models.Pool{
		Description: &d3,
		MinSession:  60,
		MaxSession:  7201,
	}

	reqBody, err = json.Marshal(p3)
	assert.NoError(t, err)
	req, err = http.NewRequest("POST", host+"/api/v1/pools", bytes.NewBuffer(reqBody))
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	pid = models.ID{}
	err = json.Unmarshal(body, &pid)
	assert.NoError(t, err)

	pid3 := *pid.ID
	_, err = uuid.Parse(pid3)
	assert.NoError(t, err)

	// Add pool to group ...

	ids := &models.IDList{pid3}

	reqBody, err = json.Marshal(ids)
	assert.NoError(t, err)
	req, err = http.NewRequest("POST", host+"/api/v1/groups/"+gid+"/pools", bytes.NewBuffer(reqBody))
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		fmt.Println("2", string(body))
	}

	pids := models.IDList{}
	err = json.Unmarshal(body, &pids)
	assert.NoError(t, err)

	assert.True(t, util.SortCompare([]string{pid0, pid1, pid3}, pids))

	// Get store....

	req, err = http.NewRequest("GET", host+"/api/v1/admin/poolstore", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	export := &models.Poolstore{}
	err = json.Unmarshal(body, export)
	assert.NoError(t, err)

	poolBytes, err := base64.StdEncoding.DecodeString(*export.Pool)
	assert.NoError(t, err)

	bookingBytes, err := base64.StdEncoding.DecodeString(*export.Booking)
	assert.NoError(t, err)

	exportedPool := &pool.PoolStore{}

	err = json.Unmarshal(poolBytes, exportedPool)
	assert.NoError(t, err)

	exportedBooking := &bookingstore.Limit{}

	err = json.Unmarshal(bookingBytes, exportedBooking)
	assert.NoError(t, err)

	if veryVerbose {
		prettyPool, err := json.MarshalIndent(exportedPool, "", "\t")
		assert.NoError(t, err)
		fmt.Println(string(prettyPool))
	}

	assert.Equal(t, 3, len(exportedPool.Groups[gid].Pools))

	// Delete pool0 from the group (but keep the pool!!)

	ids = &models.IDList{pid0}

	reqBody, err = json.Marshal(ids)
	assert.NoError(t, err)
	req, err = http.NewRequest("DELETE", host+"/api/v1/groups/"+gid+"/pools", bytes.NewBuffer(reqBody))
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		fmt.Println("3", string(body))
	}

	pids = models.IDList{}
	err = json.Unmarshal(body, &pids)
	assert.NoError(t, err)

	assert.True(t, util.SortCompare([]string{pid1, pid3}, pids))

	// Check ... 2 pools in group, 3 pools in total still

	req, err = http.NewRequest("GET", host+"/api/v1/admin/poolstore", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	export = &models.Poolstore{}
	err = json.Unmarshal(body, export)
	assert.NoError(t, err)

	poolBytes, err = base64.StdEncoding.DecodeString(*export.Pool)
	assert.NoError(t, err)

	bookingBytes, err = base64.StdEncoding.DecodeString(*export.Booking)
	assert.NoError(t, err)

	exportedPool = &pool.PoolStore{}

	err = json.Unmarshal(poolBytes, exportedPool)
	assert.NoError(t, err)

	exportedBooking = &bookingstore.Limit{}

	err = json.Unmarshal(bookingBytes, exportedBooking)
	assert.NoError(t, err)

	if veryVerbose {
		prettyPool, err := json.MarshalIndent(exportedPool, "", "\t")
		assert.NoError(t, err)
		fmt.Println(string(prettyPool))
	}

	assert.Equal(t, 2, len(exportedPool.Groups[gid].Pools))
	assert.Equal(t, 3, len(exportedPool.Pools))

	gpids := []string{}

	for _, p := range exportedPool.Groups[gid].Pools {
		gpids = append(gpids, p.ID)
	}

	assert.True(t, util.SortCompare([]string{pid1, pid3}, gpids))

	// Now replace pid1, pid3 with pid0, pid3

	ids = &models.IDList{pid0, pid3}

	reqBody, err = json.Marshal(ids)
	assert.NoError(t, err)
	req, err = http.NewRequest("PUT", host+"/api/v1/groups/"+gid+"/pools", bytes.NewBuffer(reqBody))
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		fmt.Println("4", string(body))
	}

	pids = models.IDList{}
	err = json.Unmarshal(body, &pids)
	assert.NoError(t, err)

	assert.True(t, util.SortCompare([]string{pid0, pid3}, pids))

	// Check ... 2 pools in group, 3 pools in total still
	// but that pools in group have swapped as required

	req, err = http.NewRequest("GET", host+"/api/v1/admin/poolstore", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	export = &models.Poolstore{}
	err = json.Unmarshal(body, export)
	assert.NoError(t, err)

	poolBytes, err = base64.StdEncoding.DecodeString(*export.Pool)
	assert.NoError(t, err)

	bookingBytes, err = base64.StdEncoding.DecodeString(*export.Booking)
	assert.NoError(t, err)

	exportedPool = &pool.PoolStore{}

	err = json.Unmarshal(poolBytes, exportedPool)
	assert.NoError(t, err)

	exportedBooking = &bookingstore.Limit{}

	err = json.Unmarshal(bookingBytes, exportedBooking)
	assert.NoError(t, err)

	if veryVerbose {
		prettyPool, err := json.MarshalIndent(exportedPool, "", "\t")
		assert.NoError(t, err)
		fmt.Println(string(prettyPool))
	}

	assert.Equal(t, 2, len(exportedPool.Groups[gid].Pools))
	assert.Equal(t, 3, len(exportedPool.Pools))

	gpids = []string{}

	for _, p := range exportedPool.Groups[gid].Pools {
		gpids = append(gpids, p.ID)
	}

	assert.True(t, util.SortCompare([]string{pid0, pid3}, gpids))

	// Delete pool1 altogether
	req, err = http.NewRequest("DELETE", host+"/api/v1/pools/"+pid1, nil)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	if resp.StatusCode != http.StatusNotFound {
		fmt.Println("5", string(body))
	}

	// Check ... 2 pools in group, are same 2 pools in total now that p1 is gone

	req, err = http.NewRequest("GET", host+"/api/v1/admin/poolstore", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	export = &models.Poolstore{}
	err = json.Unmarshal(body, export)
	assert.NoError(t, err)

	poolBytes, err = base64.StdEncoding.DecodeString(*export.Pool)
	assert.NoError(t, err)

	bookingBytes, err = base64.StdEncoding.DecodeString(*export.Booking)
	assert.NoError(t, err)

	exportedPool = &pool.PoolStore{}

	err = json.Unmarshal(poolBytes, exportedPool)
	assert.NoError(t, err)

	exportedBooking = &bookingstore.Limit{}

	err = json.Unmarshal(bookingBytes, exportedBooking)
	assert.NoError(t, err)

	if veryVerbose {
		prettyPool, err := json.MarshalIndent(exportedPool, "", "\t")
		assert.NoError(t, err)
		fmt.Println(string(prettyPool))
	}

	assert.Equal(t, 2, len(exportedPool.Groups[gid].Pools))
	assert.Equal(t, 2, len(exportedPool.Pools))

	gpids = []string{}

	for _, p := range exportedPool.Groups[gid].Pools {
		gpids = append(gpids, p.ID)
	}

	assert.True(t, util.SortCompare([]string{pid0, pid3}, gpids))

	// create an activity for pool0

	exp := float64(mocktime + 50)
	name4 := "act"
	thisType4 := "activity"

	d4 := models.Description{
		Name: &name4,
		Type: &thisType4,
	}

	ma := models.Activity{
		Description: &d4,
		Exp:         &exp,
	}

	reqBody, err = json.Marshal(ma)
	assert.NoError(t, err)
	req, err = http.NewRequest("POST", host+"/api/v1/pools/"+pid0+"/activities", bytes.NewBuffer(reqBody))
	assert.NoError(t, err)
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "{\"code\":602,\"message\":\"streams in body is required\"}", string(body))

	what := "data"
	url := "http://some.io"
	aud := "https://example.com"
	ct := "session"
	topic := "foo"
	scopes := []string{"read"}

	perm := models.Permission{
		Audience:       &aud,
		ConnectionType: &ct,
		Topic:          &topic,
		Scopes:         scopes,
	}
	ma.Streams = []*models.Stream{
		&models.Stream{
			For:        &what,
			URL:        &url,
			Permission: &perm,
		},
	}

	name5 := "some UI"
	thisType5 := "UI"
	url2 := "http:/some2.io"
	d5 := models.Description{
		Name: &name5,
		Type: &thisType5,
	}

	ma.Uis = []*models.UserInterface{
		&models.UserInterface{
			Description: &d5,
			URL:         &url2,
		},
	}
	reqBody, err = json.Marshal(ma)
	assert.NoError(t, err)
	req, err = http.NewRequest("POST", host+"/api/v1/pools/"+pid0+"/activities", bytes.NewBuffer(reqBody))
	assert.NoError(t, err)
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	pid = models.ID{}
	err = json.Unmarshal(body, &pid)
	assert.NoError(t, err)

	aid4 := *pid.ID
	_, err = uuid.Parse(aid4)
	assert.NoError(t, err)

	// Get store status and check for activity
	req, err = http.NewRequest("GET", host+"/api/v1/admin/status", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	ms = &models.StoreStatus{}
	err = json.Unmarshal(body, ms)
	assert.NoError(t, err)

	if veryVerbose {
		pretty, err = json.MarshalIndent(ms, "", "\t")
		fmt.Println(string(pretty))
	}

	assert.Equal(t, int64(1), ms.Activities)
	assert.Equal(t, int64(0), ms.Bookings)
	assert.Equal(t, int64(1), ms.Groups)
	assert.Equal(t, int64(2), ms.Pools)

	// check the activity exists ok ...

	req, err = http.NewRequest("GET", host+"/api/v1/admin/poolstore", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	export = &models.Poolstore{}
	err = json.Unmarshal(body, export)
	assert.NoError(t, err)

	poolBytes, err = base64.StdEncoding.DecodeString(*export.Pool)
	assert.NoError(t, err)

	bookingBytes, err = base64.StdEncoding.DecodeString(*export.Booking)
	assert.NoError(t, err)

	exportedPool = &pool.PoolStore{}

	err = json.Unmarshal(poolBytes, exportedPool)
	assert.NoError(t, err)

	exportedBooking = &bookingstore.Limit{}

	err = json.Unmarshal(bookingBytes, exportedBooking)
	assert.NoError(t, err)

	if veryVerbose {
		prettyPool, err := json.MarshalIndent(exportedPool, "", "\t")
		assert.NoError(t, err)
		fmt.Println(string(prettyPool))
	}

	// now delete it ...
	req, err = http.NewRequest("DELETE", host+"/api/v1/pools/"+pid0+"/activities/"+aid4, nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	// check no activity...
	// Get store status and check for activity
	req, err = http.NewRequest("GET", host+"/api/v1/admin/status", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	ms = &models.StoreStatus{}
	err = json.Unmarshal(body, ms)
	assert.NoError(t, err)

	if veryVerbose {
		pretty, err = json.MarshalIndent(ms, "", "\t")
		fmt.Println(string(pretty))
	}

	assert.Equal(t, int64(0), ms.Activities)
	assert.Equal(t, int64(0), ms.Bookings)
	assert.Equal(t, int64(1), ms.Groups)
	assert.Equal(t, int64(2), ms.Pools)

	// now delete group
	req, err = http.NewRequest("DELETE", host+"/api/v1/groups/"+gid, nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	// Get store status and check group is gone but pools stay
	req, err = http.NewRequest("GET", host+"/api/v1/admin/status", nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", adminBearer)
	resp, err = client.Do(req)
	assert.NoError(t, err)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	ms = &models.StoreStatus{}
	err = json.Unmarshal(body, ms)
	assert.NoError(t, err)

	if veryVerbose {
		pretty, err = json.MarshalIndent(ms, "", "\t")
		fmt.Println(string(pretty))
	}

	assert.Equal(t, int64(0), ms.Activities)
	assert.Equal(t, int64(0), ms.Bookings)
	assert.Equal(t, int64(0), ms.Groups)
	assert.Equal(t, int64(2), ms.Pools)
}
