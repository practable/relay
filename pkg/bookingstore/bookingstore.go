// Package limit supports enforcing the maximum number of
// simultaneous time-limited resources by user

package bookingstore

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/relay/pkg/booking/models"
)

// TODO - add periodic cleaning to avoid memory leak from single-use users
type confirmation struct {
	userID    string
	sessionID string
	activity  *models.Activity
}

type Limit struct {
	*sync.Mutex

	// activities maps user id to map of session ids and activity pointers
	activities map[string]map[string]*models.Activity

	// sessions maps user id to map of session ids and expiry times
	sessions map[string]map[string]int64

	// maximum number of open sessions
	max int

	flushInterval time.Duration

	provisionalPeriod time.Duration

	register chan confirmation

	ctx context.Context

	Now func() int64
}

func (l *Limit) handleRegister() {
	log.WithFields(log.Fields{"source": "bookingstore", "event": "confirm:start"}).Trace("bookstore:confirm:start")
	for {
		select {
		case <-l.ctx.Done():
			log.WithFields(log.Fields{"source": "bookingstore", "event": "confirm:stop"}).Trace("bookingstore:confirm:stop")
			return
		case c := <-l.register:
			if c.activity == nil {
				lf := log.Fields{
					"source":     "bookingstore",
					"event":      "confirm:activity:nil",
					"userID":     c.userID,
					"sessionID":  c.sessionID,
					"activityID": "",
				}
				log.WithFields(lf).Trace("bookingstore:confirm:activity:nil")
				continue
			}
			if _, ok := l.activities[c.userID]; !ok {
				l.activities[c.userID] = make(map[string]*models.Activity)
			}
			lau := l.activities[c.userID]
			lau[c.sessionID] = c.activity
			l.activities[c.userID] = lau
			ID := ""

			if c.activity.Description != nil {
				ID = c.activity.Description.ID
			}
			lf := log.Fields{
				"source":     "bookingstore",
				"event":      "confirm:activity:nil",
				"userID":     c.userID,
				"sessionID":  c.sessionID,
				"activityID": ID,
			}
			log.WithFields(lf).Trace("bookingstore:confirm:activity:ok")
		}
	}

}

// New creates a new Limit with optional
// hourly flushing to avoid memory leakage
func New(ctx context.Context) *Limit {
	l := &Limit{
		Mutex:             &sync.Mutex{},
		activities:        make(map[string]map[string]*models.Activity),
		sessions:          make(map[string]map[string]int64),
		max:               2,
		flushInterval:     time.Hour,
		provisionalPeriod: time.Hour,
		register:          make(chan confirmation),
		ctx:               ctx,
		Now:               func() int64 { return time.Now().Unix() },
	}

	go l.handleRegister()

	return l
}

func (l *Limit) WithFlush(interval time.Duration) *Limit {

	log.Trace("limit.WithFlush(): setting up")

	l.Lock()
	defer l.Unlock()

	l.flushInterval = interval

	go func() {
		log.Trace("limit.WithFlush(): starting")
		for {
			select {
			case <-l.ctx.Done():
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

func (l *Limit) WithProvisionalPeriod(interval time.Duration) *Limit {

	log.Tracef("limit.WithProvisionalPeriod(): set interval to %v", interval)

	l.Lock()
	defer l.Unlock()

	l.provisionalPeriod = interval

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

func GetSessionActivities() {
	panic("NOT IMPLEMENTED")
}

func (l *Limit) GetSessionCount(user string) int {

	sessions, ok := l.sessions[user]

	if !ok {
		return 0
	}

	return len(sessions)
}

// Flushall removes all stale entries
func (l *Limit) FlushAll() {
	l.Lock()
	defer l.Unlock()
	sessions := l.sessions

	for who, stale := range sessions {

		// update session ids to be fresh only
		fresh := l.flush(stale)
		l.sessions[who] = fresh

		lf := log.Fields{
			"source":       "bookingstore",
			"event":        "flush:sessions",
			"stale":        stale,
			"fresh":        fresh,
			"userID":       who,
			"countRemoved": len(stale) - len(fresh),
		}
		log.WithFields(lf).Trace("bookingstore:flush:sessions")

		// make fresh map of user activities
		// and copy in current activities,
		// replacing old map
		if _, ok := l.activities[who]; !ok {
			// no activity details stored for this user
			continue
		}

		la := l.activities[who]                 //old map
		ua := make(map[string]*models.Activity) //new map
		for session, _ := range fresh {
			if a, ok := la[session]; ok {

				ua[session] = a

				lf := log.Fields{
					"source":  "bookingstore",
					"session": session,
					"who":     who,
					"event":   "flush:activities:keep:ok",
				}
				log.WithFields(lf).Trace("flush:activities:keep:ok")

			} else {
				lf := log.Fields{"source": "bookingstore",
					"session": session,
					"who":     who,
					"event":   "flush:activities:keep:empty",
				}
				log.WithFields(lf).Trace("bookingstore:activities:keep:empty")
			}
		}
		l.activities[who] = la
	}
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
	return fresh
}

func (l *Limit) confirm(confirm chan struct{}, userID, sessionID string) func(activity *models.Activity) {

	return func(activity *models.Activity) {

		l.register <- confirmation{
			userID:    userID,
			sessionID: sessionID,
			activity:  activity,
		}

		close(confirm)

		lf := log.Fields{
			"source":    "bookingstore",
			"event":     "request:provisional:confirm:confirmBooking",
			"who":       userID,
			"sessionID": sessionID,
		}
		log.WithFields(lf).Trace("bookingstore:request:provisional:confirm:confirmBooking")
	}
}

func (l *Limit) autoDelete(cancel, confirm chan struct{}, userID, sessionID string) {

	cancelFunc := func() {

		l.Lock()
		defer l.Unlock()

		lf := log.Fields{
			"source":    "bookingstore",
			"event":     "request:provisional:autoDelete:deleteBooking",
			"who":       userID,
			"sessionID": sessionID,
		}

		sessions := l.sessions

		s, ok := sessions[userID]

		if !ok {
			log.WithFields(lf).Trace("bookingstore:request:provisional:autoDelete:deleteBooking:error:noSession")
			return
		}

		delete(s, sessionID)

		sessions[userID] = s

		l.sessions = sessions

		log.WithFields(lf).Trace("bookingstore:request:provisional:autoDelete:deleteBooking:ok")
	}

	select {

	case <-cancel:
		cancelFunc()
		return

	case <-time.After(l.provisionalPeriod):
		// prevent leakage if api handler stalls before
		// confirming or cancelling
		cancelFunc()
		return

	case <-l.ctx.Done():
		// done because server shut down
		lf := log.Fields{
			"source":    "bookingstore",
			"event":     "request:provisional:autoDelete:contextCancelledBeforeDecision",
			"who":       userID,
			"sessionID": sessionID,
		}
		log.WithFields(lf).Trace("bookingstore:request:provisional:autoDelete:contextCancelledBeforeDecision")
		return

	case <-confirm:
		// prevents the autodelete
		// we register on separate channel to
		// save double handling the activity pointer
		lf := log.Fields{
			"source":    "bookingstore",
			"event":     "request:provisional:autoDelete:keepBooking",
			"who":       userID,
			"sessionID": sessionID,
		}
		log.WithFields(lf).Trace("bookingstore:request:provisional:autoDelete:keepBooking")
		return
	}
}

// ProvisionalRequest checks if a user has spare capacity within their limit
// adding their request provisionally (with a delayed autodelete) if approved. The autodelete can be cancelled
// with the returned CancelFunc, in order to make the booking. If there is no quota left, an error is returned
func (l *Limit) ProvisionalRequest(userID string, exp int64) (func(), func(activity *models.Activity), error) {
	l.Lock()
	defer l.Unlock()

	// no sessions allowed?
	if l.max < 1 {
		lf := log.Fields{
			"source": "bookingstore",
			"event":  "request:provisional:granted",
			"userID": userID,
			"exp":    exp,
		}
		log.WithFields(lf).Debug("bookingstore:request:provisional:denied:noNewSessionsAllowed")
		return nil, nil, errors.New("denied: no new sessions allowed")
	}

	confirm := make(chan struct{})
	cancel := make(chan struct{})

	cancelFunc := func() { close(cancel) }

	// modify copy of map, then update main
	sessions := l.sessions
	s, ok := sessions[userID]

	sessionID := uuid.New().String()

	if !ok { // first current session, within limit, granted

		fresh := make(map[string]int64)
		fresh[sessionID] = exp
		sessions[userID] = fresh
		l.sessions = sessions

		lf := log.Fields{
			"source":       "bookingstore",
			"event":        "request:provisional:granted",
			"userID":       userID,
			"exp":          exp,
			"sessionID":    sessionID,
			"sessionCount": len(fresh),
			"max":          l.max,
		}
		log.WithFields(lf).Debug("bookingstore:request:provisional:granted")
		go l.autoDelete(cancel, confirm, userID, sessionID)
		return cancelFunc, l.confirm(confirm, userID, sessionID), nil
	}

	// flush stale sessions before checking number of current sessions
	s = l.flush(s)

	// if over limit ....
	if len(s) >= l.max {

		sessions[userID] = s
		l.sessions = sessions

		lf := log.Fields{
			"source":       "bookingstore",
			"event":        "request:provisional:granted",
			"userID":       userID,
			"exp":          exp,
			"sessionID":    sessionID,
			"sessionCount": len(s),
			"max":          l.max,
		}
		log.WithFields(lf).Debug("bookingstore:request:provisional:denied:overLimit")
		return nil, nil, errors.New("denied: over limit")
	}

	// if get here, then under limit
	s[sessionID] = exp   // s is specific to user, so this is map of sessionID to expiry
	sessions[userID] = s //put map of sessions into map by userID
	l.sessions = sessions

	lf := log.Fields{
		"source":       "bookingstore",
		"event":        "request:provisional:granted",
		"userID":       userID,
		"exp":          exp,
		"sessionID":    sessionID,
		"sessionCount": len(s),
		"max":          l.max,
	}
	log.WithFields(lf).Debug("bookingstore:request:provisional:granted")

	go l.autoDelete(cancel, confirm, userID, sessionID)
	return cancelFunc, l.confirm(confirm, userID, sessionID), nil
}

func (l *Limit) Request(who string, exp int64) bool {

	_, confirm, err := l.ProvisionalRequest(who, exp)

	if err != nil {
		return false
	}

	confirm(nil)
	return true
}
