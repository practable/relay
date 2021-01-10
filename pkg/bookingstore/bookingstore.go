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
	expiresAt int64
}

type Limit struct {
	*sync.Mutex

	// activities represents users' activities
	// it maps each user id to a map of their sessionIDs and activities
	activities map[string]map[string]*models.Activity

	// sessions represents the expiry time of current sessions
	// it maps user id to map of session ids and expiry times
	sessions map[string]map[string]int64

	// activityBySession represents all activities
	// use for reverse look up activity by sessionID
	activityBySession map[string]*models.Activity

	// userBySession represents all users
	// sue for reverse look up user by sessionID
	userBySession map[string]string

	// lastFlush represents the last time a FlushAll() was done
	// There is no point in flushing more often than the clock granularity
	// of one second
	lastFlush int64

	// lastBookingEnds represents the expiry time of the longest running session
	// when new bookings are confirmed, this is updated
	lastBookingEnds int64

	// max represents the per-user maximum number of concurrent sessions
	// there is no limit on historical usage within this package
	max int

	// lockBookings represents whether there is a lock on bookings
	// locking bookings prevents new bookings, letting
	// existing bookings continue
	lockBookings bool

	// flusInterval represents the time delay between automated FlushAll calls
	// these are intended to prevent inadvertent memory leakage
	flushInterval time.Duration

	// provisionalPeriod represents the time that a booking can remain unresolved
	// into a cancellation or a confirmation with an activity
	// failing to either cancel or confirm, results in the user being treated
	// as if they confirmed the booking, in terms of quota, but they cannot
	// access the activity details again.
	provisionalPeriod time.Duration

	// register represents the channel over which booking confirmations are registered
	register chan confirmation

	// ctx representst the cancellable context for the whole instance
	ctx context.Context

	// Now represents a function that returns the current time (mockable for non-timer parts of
	// of the package)
	Now func() int64
}

// handleRegister handles booking confirmations and should be run in one separate goro
func (l *Limit) handleRegister() {
	log.WithFields(log.Fields{"source": "bookingstore", "event": "confirm:start"}).Trace("bookstore:confirm:start")
	for {
		select {
		case <-l.ctx.Done():
			log.WithFields(log.Fields{"source": "bookingstore", "event": "confirm:stop"}).Trace("bookingstore:confirm:stop")
			return
		case c := <-l.register:

			if c.expiresAt > l.lastBookingEnds {
				l.lastBookingEnds = c.expiresAt
			}

			l.userBySession[c.sessionID] = c.userID

			if c.activity == nil {
				lf := log.Fields{
					"source":     "bookingstore",
					"event":      "confirm:activity:nil",
					"userID":     c.userID,
					"sessionID":  c.sessionID,
					"expiresAt":  c.expiresAt,
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

			l.activityBySession[c.sessionID] = c.activity

			lf := log.Fields{
				"source":     "bookingstore",
				"event":      "confirm:activity:nil",
				"userID":     c.userID,
				"sessionID":  c.sessionID,
				"expiresAt":  c.expiresAt,
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
		activityBySession: make(map[string]*models.Activity),
		userBySession:     make(map[string]string),
		max:               2,
		flushInterval:     time.Hour,
		provisionalPeriod: time.Hour,
		register:          make(chan confirmation),
		lockBookings:      false,
		ctx:               ctx,
		lastBookingEnds:   time.Now().Unix(),
		Now:               func() int64 { return time.Now().Unix() },
	}

	go l.handleRegister()

	return l
}

// WithFlush adds a periodic flush to avoid memory leakage
// Use this in production
func (l *Limit) WithFlush(interval time.Duration) *Limit {

	lf := log.Fields{
		"source":   "bookingstore",
		"event":    "withFlush:configure",
		"interval": interval,
	}
	log.WithFields(lf).Trace("bookingstore:withFlush:configure")

	l.Lock()
	defer l.Unlock()

	l.flushInterval = interval

	go func() {

		lf := log.Fields{
			"source": "bookingstore",
			"event":  "withFlush:start",
		}
		log.WithFields(lf).Trace("bookingstore:withFlush:start")

		for {
			select {
			case <-l.ctx.Done():
				lf := log.Fields{
					"source": "bookingstore",
					"event":  "withFlush:stop",
				}
				log.WithFields(lf).Trace("bookingstore:withFlush:stop")
				return
			case <-time.After(l.flushInterval):
				lf := log.Fields{
					"source": "bookingstore",
					"event":  "withFlush:flushAll",
				}
				log.WithFields(lf).Trace("bookingstore:withFlush:flushAll")
				l.FlushAll()
			}
		}
	}()

	return l
}

// WithProvisionalPeriod sets the period within which a booking can be cancelled, or confirmed
// This is used to prevent leakage of the autoDelete goroutine
func (l *Limit) WithProvisionalPeriod(interval time.Duration) *Limit {

	log.Tracef("limit.WithProvisionalPeriod(): set interval to %v", interval)

	l.Lock()
	defer l.Unlock()

	l.provisionalPeriod = interval

	return l
}

// WithMax sets the maximum number of concurrent sessions a user can have
func (l *Limit) WithMax(max int) *Limit {
	l.Lock()
	defer l.Unlock()
	l.max = max
	return l
}

// WithNow sets the time function, useful for testing basic functionality
// but does not affect anything running off timers
func (l *Limit) WithNow(now func() int64) *Limit {
	l.Lock()
	defer l.Unlock()
	l.Now = now
	return l
}

func (l *Limit) LockBookings() {
	l.Lock()
	defer l.Unlock()

	l.lockBookings = true
}

func (l *Limit) UnlockBookings() {
	l.Lock()
	defer l.Unlock()

	l.lockBookings = false
}

func (l *Limit) GetLockBookings() bool {
	l.Lock()
	defer l.Unlock()

	return l.lockBookings
}

// GetUserActivities provides pointers to all of a users activities, so that
// they can be provided with their current bookings (intended for both user and admin scope)
func (l *Limit) GetUserActivities(user string) (map[string]*models.Activity, error) {
	l.Lock()
	defer l.Unlock()

	var err error

	a, ok := l.activities[user]
	if !ok {
		err = errors.New("not found")
	}

	return a, err
}

// GetAllActivities returns a map of all activities by session
// This must be for admin use only - else anyone can use anyone else's
// sessions - activities contain credentials
func (l *Limit) GetAllActivities() map[string]*models.Activity {
	l.Lock()
	defer l.Unlock()

	return l.activityBySession
}

func (l *Limit) GetAllActivitiesCount() int {
	l.Lock()
	defer l.Unlock()

	return len(l.activityBySession)
}

// GetLastBookingEnds is an admin function to help figure out
// how long until the system is free of booked sessions
// e.g. after disabling new bookings, existing bookings
// will still have some time to run
func (l *Limit) GetLastBookingEnds() int64 {
	l.Lock()
	defer l.Unlock()

	return l.lastBookingEnds
}

// Get all sessionCount is primarily an admin function
// for a bit of dashboard eye-candy, but there is no reason
// that users can't see this too
func (l *Limit) GetAllSessionCount() int {
	l.FlushAll()
	l.Lock()
	defer l.Unlock()
	return len(l.activityBySession)
}

// GetActivityFromSessionID allows an activity to be retrieved even if the
// userID has been lost -> admin scope only!
func (l *Limit) GetActivityFromSessionID(sid string) (*models.Activity, error) {
	l.FlushAll()
	l.Lock()
	defer l.Unlock()

	var err error

	a, ok := l.activityBySession[sid]

	if !ok {
		err = errors.New("not found")
	}

	return a, err
}

// GetUserSessionCount returns number of current sessions a user has
func (l *Limit) GetUserSessionCount(user string) int {
	l.Lock()
	defer l.Unlock()

	l.flushUser(user)
	return len(l.sessions[user])
}

// FlushAll removes all stale entries from all maps
func (l *Limit) FlushAll() {
	l.Lock()
	defer l.Unlock()

	timeSinceLastFlush := time.Now().Unix() - l.lastFlush

	if timeSinceLastFlush < 1 { // nothing has changed within the last second.
		return
	}

	sessions := l.sessions
	for userID, _ := range sessions {
		l.flushUser(userID)
	}

	l.lastFlush = time.Now().Unix()
}

// flushUser removes stale entries from all maps
// for a given user
func (l *Limit) flushUser(user string) {

	stale := l.sessions[user]
	now := l.Now()
	for k, s := range stale {
		if s < now {
			delete(l.sessions[user], k)
			delete(l.activities[user], k)
			delete(l.activityBySession, k)
			delete(l.userBySession, k)
		}
	}

	removed := len(stale) - len(l.sessions[user])

	if removed > 0 {
		lf := log.Fields{
			"source":       "bookingstore",
			"event":        "flush:sessions:removed",
			"userID":       user,
			"countRemoved": removed,
		}
		log.WithFields(lf).Trace("bookingstore:flush:sessions:removed")
	}

	return
}

// confim registers the activity details over the confirm channel, if called by the booker
// (is returned when a provisional session is requested)
func (l *Limit) confirm(confirm chan struct{}, userID, sessionID string, expiresAt int64) func(activity *models.Activity) {

	return func(activity *models.Activity) {

		l.register <- confirmation{
			userID:    userID,
			sessionID: sessionID,
			activity:  activity,
			expiresAt: expiresAt,
		}

		close(confirm)

		lf := log.Fields{
			"source":    "bookingstore",
			"event":     "request:provisional:confirm:confirmBooking",
			"userID":    userID,
			"sessionID": sessionID,
			"expiresAt": expiresAt,
		}
		log.WithFields(lf).Trace("bookingstore:request:provisional:confirm:confirmBooking")
	}
}

// autoDelete handles the cancellation of a provisional session
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
// If so, a provisional booking is made, and cancel and confirm functions are returned
// along with the sessionID
// The session should either be cancelled, or confirmed with argument to models.Activity so that
// booking details can be retrieved again later in the booking if need be, from this store.
// If there is no quota left, or new bookings are suspended, an error is returned
func (l *Limit) ProvisionalRequest(userID string, exp int64) (func(), func(activity *models.Activity), string, error) {
	l.Lock()
	defer l.Unlock()

	if exp < time.Now().Unix() {
		lf := log.Fields{
			"source": "bookingstore",
			"event":  "request:provisional:denied:sessionExpiresInPast",
			"userID": userID,
			"exp":    exp,
		}
		log.WithFields(lf).Debug("bookingstore:request:provisional:denied:sessionExpiresInPast")
		return nil, nil, "", errors.New("denied: session expires in past")
	}

	// no sessions allowed?
	if l.max < 1 || l.lockBookings {
		lf := log.Fields{
			"source": "bookingstore",
			"event":  "request:provisional:denied:noNewSessionsAllowed",
			"userID": userID,
			"exp":    exp,
		}
		log.WithFields(lf).Debug("bookingstore:request:provisional:denied:noNewSessionsAllowed")
		return nil, nil, "", errors.New("denied: no new sessions allowed")
	}

	confirm := make(chan struct{})
	cancel := make(chan struct{})

	cancelFunc := func() { close(cancel) }

	sessionID := uuid.New().String()

	// check if user has a session map
	_, ok := l.sessions[userID]

	if !ok {
		l.sessions[userID] = make(map[string]int64)
	} else {
		l.flushUser(userID) //up date our session list
	}

	// user session count
	usc := len(l.sessions[userID])

	if usc >= l.max {

		lf := log.Fields{
			"source":           "bookingstore",
			"event":            "request:provisional:denied:overLimit",
			"userID":           userID,
			"exp":              exp,
			"sessionID":        sessionID,
			"userSessionCount": usc,
			"max":              l.max,
		}
		log.WithFields(lf).Debug("bookingstore:request:provisional:denied:overLimit")
		return nil, nil, "", errors.New("denied: over limit")
	}

	// grant provisional session

	l.sessions[userID][sessionID] = exp

	lf := log.Fields{
		"source":           "bookingstore",
		"event":            "request:provisional:granted",
		"userID":           userID,
		"exp":              exp,
		"sessionID":        sessionID,
		"userSessionCount": usc,
		"max":              l.max,
	}
	log.WithFields(lf).Debug("bookingstore:request:provisional:granted")

	go l.autoDelete(cancel, confirm, userID, sessionID)
	return cancelFunc, l.confirm(confirm, userID, sessionID, exp), sessionID, nil
}

// Request is primarily for testing purposes but may be useful in some other
// context so remains exported
func (l *Limit) Request(who string, exp int64) (string, error) {

	_, confirm, ID, err := l.ProvisionalRequest(who, exp)

	if err == nil {
		confirm(nil)
	}

	return ID, err
}
