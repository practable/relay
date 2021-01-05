package limit

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
		log.SetLevel(log.DebugLevel)
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

	ctx, cancel := context.WithCancel(context.Background())
	l := New().WithFlush(ctx, time.Second)
	assert.True(t, l.Request("user0-TestWithFlush", l.Now()+1))
	assert.Equal(t, 1, len(l.sessions))
	time.Sleep(2 * time.Second)
	assert.Equal(t, 0, len(l.sessions))
	assert.True(t, l.Request("user1-TestWithFlush", l.Now()+1))
	assert.Equal(t, 1, len(l.sessions))
	cancel()
	time.Sleep(2 * time.Second)
	assert.Equal(t, 1, len(l.sessions))
	// no flush after cancel (i.e does not leak flush goroutine)

}

func mockTime(now *int64) int64 {
	return *now
}

func TestNewHitLimit(t *testing.T) {

	t.Parallel()

	mocktime := time.Now().Unix()
	t0 := mocktime

	l := New().WithNow(func() int64 { return func(now *int64) int64 { return *now }(&mocktime) })

	assert.True(t, l.Request("user0-TestNewHitLimit", t0+300))
	assert.Equal(t, 1, len(l.sessions))
	assert.True(t, l.Request("user0-TestNewHitLimit", t0+600))
	assert.Equal(t, 1, len(l.sessions)) //one user
	assert.False(t, l.Request("user0-TestNewHitLimit", t0+600))
	assert.Equal(t, 1, len(l.sessions)) //one user
	mocktime = t0 + 400                 //first session expired
	assert.True(t, l.Request("user0-TestNewHitLimit", t0+600))
	assert.Equal(t, 1, len(l.sessions)) //one user
	assert.True(t, l.Request("user1-TestNewHitLimit", t0+1200))
	assert.Equal(t, 2, len(l.sessions)) //two users

}
