package bookingstore

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Setup logging
	debug := false

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

func TestNewWithFlush(t *testing.T) {

	t.Parallel()

	// have to do this test in real time due to way flush intervals
	// are worked from the system clock

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := New(ctx).WithFlush(time.Second)
	u0 := "user0-TestWithFlush"
	assert.Equal(t, 0, l.GetSessionCount(u0))

	_, err := l.Request(u0, l.Now()+1)
	assert.NoError(t, err)
	assert.Equal(t, 1, l.GetSessionCount(u0))

	// count correctly goes 1 -> 0 due to flush
	time.Sleep(2 * time.Second)
	assert.Equal(t, 0, l.GetSessionCount(u0))

	// can't test cancelling flush independently,
	// because other sub-tasks share the context

}

func mockTime(now *int64) int64 {
	return *now
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

	assert.Equal(t, 0, l.GetSessionCount(u0))

	//grant 1/2
	_, err := l.Request(u0, t0+300)
	assert.NoError(t, err)

	assert.Equal(t, 1, l.GetSessionCount(u0))

	// grant 2/2
	_, err = l.Request(u0, t0+600)
	assert.NoError(t, err)
	assert.Equal(t, 2, l.GetSessionCount(u0))

	// deny as at limit of 2
	_, err = l.Request(u0, t0+600)
	assert.Error(t, err)
	assert.Equal(t, 2, l.GetSessionCount(u0))

	// but grant another user to 1/2
	_, err = l.Request(u1, t0+600)
	assert.NoError(t, err)

	assert.Equal(t, 1, l.GetSessionCount(u1))

	// wait for first session to finish
	mocktime = t0 + 400
	l.FlushAll()
	// back to one session
	assert.Equal(t, 1, l.GetSessionCount(u0))

	//grant 2/2
	_, err = l.Request(u0, t0+600)
	assert.NoError(t, err)
	assert.Equal(t, 2, l.GetSessionCount(u0))

}

func TestProvisionalRequest(t *testing.T) {
	// a real-time test because of the way
	// intervals are timed with time.After
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := New(ctx).WithProvisionalPeriod(time.Second)
	u0 := "user0-TestProvisionalRequest"

	assert.Equal(t, 0, l.GetSessionCount(u0))

	cancelBooking, _, _, err := l.ProvisionalRequest(u0, 5)

	assert.NoError(t, err)
	assert.Equal(t, 1, l.GetSessionCount(u0))

	// cancel the booking
	cancelBooking()

	time.Sleep(10 * time.Millisecond)

	// booking gone
	assert.Equal(t, 0, l.GetSessionCount(u0))

	_, confirm, _, err := l.ProvisionalRequest(u0, 5)

	assert.NoError(t, err)
	assert.Equal(t, 1, l.GetSessionCount(u0))

	confirm(nil)
	//booking stays
	assert.Equal(t, 1, l.GetSessionCount(u0))
	// belt and braces, time out the provisional period
	time.Sleep(2 * time.Second)
	// booking stays
	assert.Equal(t, 1, l.GetSessionCount(u0))

}
