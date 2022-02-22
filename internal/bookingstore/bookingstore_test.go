package bookingstore

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/practable/relay/internal/booking/models"
	"github.com/practable/relay/internal/util"
	"github.com/xtgo/uuid"
)

var debug bool

func TestMain(m *testing.M) {
	// Setup logging
	debug = false

	if debug {
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true, DisableColors: true})
		defer log.SetOutput(os.Stdout)

	} else {
		var ignore bytes.Buffer
		logignore := bufio.NewWriter(&ignore)
		log.SetOutput(logignore)
	}

	exitVal := m.Run()

	os.Exit(exitVal)
}

func TestMessage(t *testing.T) {

	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := New(ctx)

	defaultMsg := "Open for bookings"

	assert.Equal(t, defaultMsg, l.GetMessage())

	newMsg := "Some other message"

	l.SetMessage(newMsg)

	assert.Equal(t, newMsg, l.GetMessage())

}

func TestNewWithFlush(t *testing.T) {

	t.Parallel()

	// have to do this test in real time due to way flush intervals
	// are worked from the system clock

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := New(ctx).WithFlush(time.Second)
	u0 := "user0-TestWithFlush"
	assert.Equal(t, 0, l.GetUserSessionCount(u0))

	_, err := l.Request(u0, l.Now()+1)
	assert.NoError(t, err)
	assert.Equal(t, 1, l.GetUserSessionCount(u0))

	// count correctly goes 1 -> 0 due to flush
	time.Sleep(2 * time.Second)
	assert.Equal(t, 0, l.GetUserSessionCount(u0))

	// can't test cancelling flush independently,
	// because other sub-tasks share the context

}

func TestNewHitLimit(t *testing.T) {
	// can mock time in this test
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mocktime := time.Now().Unix()
	t0 := mocktime

	l := New(ctx).WithNow(func() int64 { return func(now *int64) int64 { return *now }(&mocktime) })
	u0 := "user0-TestNewHitLimit"
	u1 := "user1-TestNewHitLimit"

	assert.Equal(t, 0, l.GetUserSessionCount(u0))

	//grant 1/2
	_, err := l.Request(u0, t0+300)
	assert.NoError(t, err)

	assert.Equal(t, 1, l.GetUserSessionCount(u0))

	// grant 2/2
	_, err = l.Request(u0, t0+600)
	assert.NoError(t, err)
	assert.Equal(t, 2, l.GetUserSessionCount(u0))

	// deny as at limit of 2
	_, err = l.Request(u0, t0+600)
	assert.Error(t, err)
	assert.Equal(t, 2, l.GetUserSessionCount(u0))

	// but grant another user to 1/2
	_, err = l.Request(u1, t0+600)
	assert.NoError(t, err)

	assert.Equal(t, 1, l.GetUserSessionCount(u1))

	// wait for first session to finish
	mocktime = t0 + 400
	l.FlushAll()
	// back to one session
	assert.Equal(t, 1, l.GetUserSessionCount(u0))

	//grant 2/2
	_, err = l.Request(u0, t0+600)
	assert.NoError(t, err)
	assert.Equal(t, 2, l.GetUserSessionCount(u0))

}

func TestProvisionalRequest(t *testing.T) {
	// a real-time test because of the way
	// intervals are timed with time.After
	t.Parallel()

	t0 := time.Now().Unix()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := New(ctx).WithProvisionalPeriod(time.Second)
	u0 := "user0-TestProvisionalRequest"

	assert.Equal(t, 0, l.GetUserSessionCount(u0))

	cancelBooking, _, _, err := l.ProvisionalRequest(u0, t0+5)

	assert.NoError(t, err)

	assert.Equal(t, 1, l.GetUserSessionCount(u0))

	// cancel the booking
	cancelBooking()

	time.Sleep(10 * time.Millisecond)

	// booking gone
	assert.Equal(t, 0, l.GetUserSessionCount(u0))

	_, confirm, _, err := l.ProvisionalRequest(u0, t0+5)

	assert.NoError(t, err)
	assert.Equal(t, 1, l.GetUserSessionCount(u0))

	confirm(nil)
	//booking stays
	assert.Equal(t, 1, l.GetUserSessionCount(u0))
	// belt and braces, time out the provisional period
	time.Sleep(2 * time.Second)
	// booking stays
	assert.Equal(t, 1, l.GetUserSessionCount(u0))

}

func TestLockUnlockBooking(t *testing.T) {
	t.Parallel()

	t0 := time.Now().Unix()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := New(ctx)

	u0 := "user0-LockBookings"

	assert.Equal(t, 0, l.GetUserSessionCount(u0))

	cancelBooking, _, ID, err := l.ProvisionalRequest(u0, t0+5)

	assert.NoError(t, err)

	_, err = uuid.Parse(ID)

	assert.NoError(t, err)

	cancelBooking()

	time.Sleep(time.Millisecond)

	l.LockBookings()

	assert.Equal(t, 0, l.GetUserSessionCount(u0))

	_, _, _, err = l.ProvisionalRequest(u0, t0+5)

	assert.Error(t, err)

	assert.Equal(t, "denied: no new sessions allowed", err.Error())

	l.UnlockBookings()

	cancelBooking, _, ID, err = l.ProvisionalRequest(u0, t0+5)

	assert.NoError(t, err)

	_, err = uuid.Parse(ID)

	assert.NoError(t, err)

	cancelBooking()

}

func TestDenySessionExpiringInPast(t *testing.T) {
	t.Parallel()

	t0 := time.Now().Unix()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := New(ctx)

	u0 := "user0-DenySessionExpiringInPast"

	assert.Equal(t, 0, l.GetUserSessionCount(u0))

	_, _, _, err := l.ProvisionalRequest(u0, t0-5)

	assert.Error(t, err)

	assert.Equal(t, "denied: session expires in past", err.Error())

}

func TestGetUserSessionCountNoUser(t *testing.T) {

	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := New(ctx)
	u0 := "user0-does-not-exist"
	count := l.GetUserSessionCount(u0)
	assert.Equal(t, 0, count)
	count = l.GetUserSessionCount("")
	assert.Equal(t, 0, count)

}

func TestConfirmGetActivity(t *testing.T) {

	t.Parallel()

	t0 := time.Now().Unix()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := New(ctx).WithProvisionalPeriod(time.Second)
	u0 := "user0-ConfirmGetActivity"

	assert.Equal(t, 0, l.GetUserSessionCount(u0))

	_, confirm, sessionID0, err := l.ProvisionalRequest(u0, t0+5)
	assert.NoError(t, err)
	_, err = uuid.Parse(sessionID0)
	assert.NoError(t, err)
	name0 := "test activity 0"

	a0 := &models.Activity{
		Description: &models.Description{
			Name: &name0,
		},
	}
	confirm(a0)

	time.Sleep(time.Millisecond)

	am, err := l.GetUserActivities(u0)

	assert.NoError(t, err)

	a0r, ok := am[sessionID0]

	assert.True(t, ok)

	assert.Equal(t, *a0, *a0r)

	// add another activity for u0

	_, confirm, sessionID1, err := l.ProvisionalRequest(u0, t0+5)

	assert.NoError(t, err)

	_, err = uuid.Parse(sessionID1)

	assert.NoError(t, err)
	name1 := "test activity 1"

	a1 := &models.Activity{
		Description: &models.Description{
			Name: &name1,
		},
	}
	confirm(a1)

	time.Sleep(time.Millisecond)
	time.Sleep(time.Millisecond)

	am, err = l.GetUserActivities(u0)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(am))

	a1r, ok := am[sessionID1]

	assert.True(t, ok)

	assert.Equal(t, *a1, *a1r)

	if debug {
		ps, err := json.MarshalIndent(l.Sessions, "", "  ")
		assert.NoError(t, err)
		fmt.Println("---SESSIONS---")
		fmt.Println(string(ps))

		pa, err := json.MarshalIndent(l.Activities, "", "  ")
		assert.NoError(t, err)
		fmt.Println("---ACTIVITIES---")
		fmt.Println(string(pa))

		pas, err := json.MarshalIndent(l.ActivityBySession, "", "  ")
		assert.NoError(t, err)
		fmt.Println("---ACTIVITIES BY SESSION---")
		fmt.Println(string(pas))

		pus, err := json.MarshalIndent(l.UserBySession, "", "  ")
		assert.NoError(t, err)
		fmt.Println("---USERS BY SESSION---")
		fmt.Println(string(pus))

	}

	assert.NoError(t, err)
	assert.Equal(t, 2, l.GetUserSessionCount(u0))
	assert.Equal(t, 2, l.GetAllSessionCount())

	all := l.GetAllActivities()

	anames := []string{name0, name1}

	anamesr := []string{}

	for _, a := range all {
		anamesr = append(anamesr, *a.Description.Name)
	}

	assert.True(t, util.SortCompare(anames, anamesr))

	exp := l.GetLastBookingEnds()

	assert.Equal(t, t0+5, exp)

	_, err = l.GetActivityFromSessionID("not a session id")
	assert.Error(t, err)
	assert.Equal(t, "not found", err.Error())
	a1r2, err := l.GetActivityFromSessionID(sessionID1)

	assert.NoError(t, err)

	assert.Equal(t, *a1.Description.Name, *a1r2.Description.Name)

}

func TestImportExport(t *testing.T) {

	// can mock time in this test

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mocktime := time.Now().Unix()
	t0 := mocktime

	l := New(ctx).WithNow(func() int64 { return func(now *int64) int64 { return *now }(&mocktime) })
	u0 := "user0-TestNewHitLimit"
	u1 := "user1-TestNewHitLimit"

	assert.Equal(t, 0, l.GetUserSessionCount(u0))

	//grant 1/2
	_, err := l.Request(u0, t0+300)
	assert.NoError(t, err)

	assert.Equal(t, 1, l.GetUserSessionCount(u0))

	// grant 2/2
	_, err = l.Request(u0, t0+600)
	assert.NoError(t, err)
	assert.Equal(t, 2, l.GetUserSessionCount(u0))

	// import and export!

	if true {
		b, err := l.ExportAll()

		assert.NoError(t, err)

		l, err = ImportAll(l, b)

		l.SetNow(func() int64 { return func(now *int64) int64 { return *now }(&mocktime) })

		assert.NoError(t, err)

		time.Sleep(10 * time.Millisecond)
	}
	if debug {
		t.Log("servicesContextSeenByTestRoutine", l.ctxServices)
	}
	// deny as at limit of 2
	_, err = l.Request(u0, t0+600)
	assert.Error(t, err)
	assert.Equal(t, 2, l.GetUserSessionCount(u0))

	// but grant another user to 1/2
	_, err = l.Request(u1, t0+600)
	assert.NoError(t, err)

	assert.Equal(t, 1, l.GetUserSessionCount(u1))

	// wait for first session to finish
	mocktime = t0 + 400
	l.FlushAll()
	// back to one session
	assert.Equal(t, 1, l.GetUserSessionCount(u0))

	//grant 2/2
	_, err = l.Request(u0, t0+600)
	assert.NoError(t, err)
	assert.Equal(t, 2, l.GetUserSessionCount(u0))

	if debug {
		t.Log("----------TEST COMPLETED-------------")
	}

}

func TestIndividualChildContextCancellation(t *testing.T) {
	t.Parallel()
	ctxP, cancelP := context.WithCancel(context.Background())
	defer cancelP()

	ctx0, cancel0 := context.WithCancel(ctxP)

	ctx1, cancel1 := context.WithCancel(ctxP)
	defer cancel1()

	go func() {

		select {

		case <-time.After(time.Second):
			t.Error("unexpected timeout ctx0")
		case <-ctx0.Done():
		}

	}()

	go func() {

		select {

		case <-time.After(time.Second):
		case <-ctx1.Done():
			t.Error("unexpected done ctx1")
		}

	}()

	cancel0()

	time.Sleep(2 * time.Second)

}

func TestIndividualChildContextCancellationInStruct(t *testing.T) {
	t.Parallel()
	ctxP, cancelP := context.WithCancel(context.Background())
	defer cancelP()

	ctx0, cancel0 := context.WithCancel(ctxP)

	l := &Limit{}

	l.ctxServices = ctx0
	l.cancelServices = cancel0

	ctx1, cancel1 := context.WithCancel(ctxP)

	defer cancel1()

	go func() {

		select {

		case <-time.After(time.Second):
			t.Error("unexpected timeout ctx0")
		case <-ctx0.Done():
		}

	}()

	go func() {

		select {

		case <-time.After(time.Second):
		case <-ctx1.Done():
			t.Error("unexpected done ctx1")
		}

	}()

	l.cancelServices()

	time.Sleep(2 * time.Second)

}

func TestIndividualChildContextCancellationInStructSwap(t *testing.T) {

	t.Parallel()

	ctxP, cancelP := context.WithCancel(context.Background())
	defer cancelP()

	ctx0, cancel0 := context.WithCancel(ctxP)

	l := &Limit{}

	l.ctxServices = ctx0
	l.cancelServices = cancel0

	oldCancel := l.cancelServices

	ctx1, cancel1 := context.WithCancel(ctxP)
	l.ctxServices = ctx1
	l.cancelServices = cancel1

	defer cancel1()

	go func() {

		select {

		case <-time.After(time.Second):
			t.Error("unexpected timeout ctx0")
		case <-ctx0.Done():
		}

	}()

	go func() {

		select {

		case <-time.After(time.Second):
		case <-ctx1.Done():
			t.Error("unexpected done ctx1")
		}

	}()

	oldCancel()

	time.Sleep(2 * time.Second)

}
