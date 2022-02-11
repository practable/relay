// Package limit supports enforcing the maximum number of
// simultaneous time-limited resources by user
package limit

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// Limit represents a limit store
type Limit struct {
	*sync.Mutex

	// sessions maps user id to array of expiry times
	sessions map[string]map[string]int64

	// maximum number of open sessions
	max int

	flushInterval time.Duration

	provisionalPeriod time.Duration

	Now func() int64
}

// New creates a new Limit with optional
// hourly flushing to avoid memory leakage
func New() *Limit {
	l := &Limit{
		&sync.Mutex{},
		make(map[string]map[string]int64),
		2,
		time.Hour,
		time.Minute,
		func() int64 { return time.Now().Unix() },
	}

	return l
}

// WithFlush adds a regular flushing action that occurs every interval
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

// WithProvisionalPeriod specifies the provisional period for provisional bookings
func (l *Limit) WithProvisionalPeriod(interval time.Duration) *Limit {

	log.Tracef("limit.WithProvisionalPeriod(): set interval to %v", interval)

	l.Lock()
	defer l.Unlock()

	l.provisionalPeriod = interval

	return l
}

// WithMax sets the maximum number of bookings per user
func (l *Limit) WithMax(max int) *Limit {
	l.Lock()
	defer l.Unlock()
	l.max = max
	return l
}

// WithNow sets the function used to obtain the current time (required for testing)
func (l *Limit) WithNow(now func() int64) *Limit {
	l.Lock()
	defer l.Unlock()
	l.Now = now
	return l
}

// GetSessions returns the current sessions for a user
func (l *Limit) GetSessions(user string) int {

	sessions, ok := l.sessions[user]

	if !ok {
		return 0
	}

	return len(sessions)
}

// FlushAll removes all stale entries
func (l *Limit) FlushAll() {
	l.Lock()
	defer l.Unlock()
	sessions := l.sessions
	for who, stale := range sessions {
		//fresh := l.flush(stale)
		l.sessions[who] = l.flush(stale)
	}
	//l.sessions = sessions
}

// Function flush removes stale entries from an array of expiry times
// only call when you have a lock!
func (l *Limit) flush(stale map[string]int64) map[string]int64 {

	fresh := make(map[string]int64)

	now := l.Now() //get time once per run for testable performance

	for k, s := range stale {

		if s > now {
			fresh[k] = s // keep current sessions only
		}
	}

	log.WithFields(log.Fields{"now": now, "stale": stale, "fresh": fresh}).Trace("l.flush()")

	return fresh //stale is a reference so _should_ update
}

// Confirm confirms a provisional booking
func Confirm(confirm chan<- struct{}) func() {
	return func() {
		close(confirm)
	}
}

// ProvisionalRequest checks if a user has spare capacity within their limit
// adding their request provisionally (with a delayed autodelete) if approved. The autodelete can be cancelled
// with the returned CancelFunc, in order to make the booking. If there is no quota left, an error is returned
func (l *Limit) ProvisionalRequest(who string, exp int64) (func(), error) {
	l.Lock()
	defer l.Unlock()

	sessions := l.sessions

	s, ok := sessions[who]

	confirm := make(chan struct{})

	id := uuid.New().String()

	if !ok { // first session

		// first session, denied
		if l.max < 1 {
			log.WithFields(log.Fields{"who": who, "exp": exp}).Debugf("l.Request(): max sessions set to zero, denied first request by %s", who)
			return nil, errors.New("no sessions allowed")
		}

		// first session, granted
		fresh := make(map[string]int64)
		fresh[id] = exp
		sessions[who] = fresh
		l.sessions = sessions

		go func() {
			select {
			case <-time.After(l.provisionalPeriod):
				l.Lock()
				defer l.Unlock()
				sessions := l.sessions
				s, ok := sessions[who]
				if !ok {
					return
				}
				delete(s, id)
				sessions[who] = s
				l.sessions = sessions
				return
			case <-confirm:
				// do nothing (this prevent autodelete)
				return

			}
		}()
		log.WithFields(log.Fields{"who": who, "exp": exp, "fresh": s}).Debugf("l.Request(%s): granted first request by %s now has %d/%d sessions", id[0:6], who, len(fresh), l.max)
		return Confirm(confirm), nil
	}
	stale := s
	s = l.flush(s)

	// at or over limit already
	if len(s) >= l.max {
		sessions[who] = s
		l.sessions = sessions
		log.WithFields(log.Fields{"who": who, "exp": exp, "fresh": s, "stale": stale}).Debugf("l.Request(%s): denied request by %s, has %d/%d sessions already", id[0:6], who, len(s), l.max)
		return nil, errors.New("denied - over limit")
	}

	// under limit
	s[id] = exp
	sessions[who] = s
	l.sessions = sessions
	log.WithFields(log.Fields{"who": who, "exp": exp, "fresh": s, "stale": stale}).Debugf("l.Request(%s): granted request by %s, now has %d/%d sessions", id[0:6], who, len(s), l.max)
	return Confirm(confirm), nil

	// Returning true does not mean the next request will be
	// granted, only that this one was.
}

// Request makes a provisional request, and immediately confirms it. Used for testing.
func (l *Limit) Request(who string, exp int64) bool {

	confirm, err := l.ProvisionalRequest(who, exp)

	if err != nil {
		return false
	}

	confirm()
	return true
}
