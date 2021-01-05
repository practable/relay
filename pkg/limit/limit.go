// Package limit supports enforcing the maximum number of
// simultaneous time-limited resources by user
package limit

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// TODO - add periodic cleaning to avoid memory leak from single-use users

type Limit struct {
	*sync.Mutex

	// sessions maps user id to array of expiry times
	sessions map[string][]int64

	// maximum number of open sessions
	max int

	flushInterval time.Duration

	Now func() int64
}

// New creates a new Limit with optional
// hourly flushing to avoid memory leakage
func New() *Limit {
	l := &Limit{
		&sync.Mutex{},
		make(map[string][]int64),
		2,
		time.Hour,
		func() int64 { return time.Now().Unix() },
	}

	return l
}

func (l *Limit) WithFlush(ctx context.Context, interval time.Duration) *Limit {

	log.Trace("limit.WithFlush(): setting up")

	l.Lock()
	defer l.Unlock()

	l.flushInterval = interval

	go func() {
		log.Trace("limit.WithFlush(): starting")
		for {
			select {
			case <-ctx.Done():
				log.Trace("limit.WithFlush(): finished because ctx.Done()")
				return
			case <-time.After(l.flushInterval):
				log.Trace("limit.WithFlush(): flushing all")
				l.FlushAll()
			}
		}
	}()

	return l
}

func (l *Limit) WithMax(max int) *Limit {
	l.Lock()
	defer l.Unlock()
	l.max = max
	return l
}

func (l *Limit) WithNow(now func() int64) *Limit {
	l.Lock()
	defer l.Unlock()
	l.Now = now
	return l
}

// Flushall removes all stale entries
func (l *Limit) FlushAll() {
	l.Lock()
	defer l.Unlock()
	sessions := l.sessions
	for user, stale := range sessions {
		fresh := l.flush(stale)
		if len(fresh) < 1 {
			delete(sessions, user)
		} else {
			sessions[user] = fresh
		}
	}
	l.sessions = sessions
}

// Function flush removes stale entries from an array of expiry times
func (l *Limit) flush(stale []int64) []int64 {
	// no mutex as only reading clock func
	// which is not intended to be re-written
	// after initialisation

	fresh := []int64{}

	now := l.Now() //get time once per run for testable performance

	for _, s := range stale {

		if s > now {
			fresh = append(fresh, s) //keep current sessions only
		}
	}

	log.WithFields(log.Fields{"now": now, "stale": stale, "fresh": fresh}).Trace("l.flush()")

	return fresh
}

// Request checks if a user has spare capacity within their limit
// adding their request if approved, and returning true;
// returns false otherwise (and does not add request)
func (l *Limit) Request(who string, exp int64) bool {
	l.Lock()
	defer l.Unlock()

	sessions := l.sessions

	stale, ok := sessions[who]

	if !ok {
		if l.max < 1 {
			log.WithFields(log.Fields{"who": who, "exp": exp}).Debugf("l.Request(): max sessions set to zero, denied first request by %s", who)
			return false // no sessions allowed
		}
		fresh := []int64{exp}
		sessions[who] = fresh
		l.sessions = sessions
		log.WithFields(log.Fields{"who": who, "exp": exp, "fresh": fresh}).Debugf("l.Request(): granted first request by %s now has %d/%d sessions", who, len(fresh), l.max)
		return true
	}

	fresh := l.flush(stale)

	// at or over limit already
	if len(fresh) >= l.max {
		sessions[who] = fresh
		l.sessions = sessions
		log.WithFields(log.Fields{"who": who, "exp": exp, "fresh": fresh, "stale": stale}).Debugf("l.Request(): denied request by %s, has %d/%d sessions already", who, len(fresh), l.max)
		return false
	}

	// under limit
	fresh = append(fresh, exp)
	sessions[who] = fresh
	l.sessions = sessions
	log.WithFields(log.Fields{"who": who, "exp": exp, "fresh": fresh, "stale": stale}).Debugf("l.Request(): granted request by %s, now has %d/%d sessions", who, len(fresh), l.max)
	return true

	// Returning true does not mean the next request will be
	// granted, only that this one was.
}
