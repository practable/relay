// Package limit supports enforcing the maximum number of
// simultaneous time-limited resources by user

package bookingstore

import (
	"context"
	"encoding/json"
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
	*sync.Mutex `json:"-"`

	// activities represents users' activities
	// it maps each user id to a map of their sessionIDs and activities
	Activities map[string]map[string]*models.Activity `json:"activities"`

	// sessions represents the expiry time of current sessions
	// it maps user id to map of session ids and expiry times
	Sessions map[string]map[string]int64 `json:"sessions"`

	// activityBySession represents all activities
	// use for reverse look up activity by sessionID
	ActivityBySession map[string]*models.Activity `json:"activityBySession"`

	// userBySession represents all users
	// sue for reverse look up user by sessionID
	UserBySession map[string]string `json:"userBySession"`

	// lastFlush represents the last time a FlushAll() was done
	// There is no point in flushing more often than the clock granularity
	// of one second
	LastFlush int64 `json:"lastFlush"`

	// lastBookingEnds represents the expiry time of the longest running session
	// when new bookings are confirmed, this is updated
	LastBookingEnds int64 `json:"lastBookingEnds"`

	// max represents the per-user maximum number of concurrent sessions
	// there is no limit on historical usage within this package
	Max int `json:"max"`

	// lockBookings represents whether there is a lock on bookings
	// locking bookings prevents new bookings, letting
	// existing bookings continue
	Locked bool `json:"lockBookings"`

	// flusInterval represents the time delay between automated FlushAll calls
	// these are intended to prevent inadvertent memory leakage
	FlushInterval time.Duration `json:"flushInterval"`

	// provisionalPeriod represents the time that a booking can remain unresolved
	// into a cancellation or a confirmation with an activity
	// failing to either cancel or confirm, results in the user being treated
	// as if they confirmed the booking, in terms of quota, but they cannot
	// access the activity details again.
	ProvisionalPeriod time.Duration `json:"provisionalPeriod"`

	// register represents the channel over which booking confirmations are registered
	register chan confirmation

	// ctx representst the cancellable context for the whole instance
	ctx context.Context

	ctxServices context.Context

	cancelServices context.CancelFunc

	// Now represents a function that returns the current time (mockable for non-timer parts of
	// of the package)
	Now func() int64 `json:"-"`
}

// New creates a new Limit with optional
// hourly flushing to avoid memory leakage
func New(ctx context.Context) *Limit {

	// value for debugging context swaps...
	ctxNew := context.WithValue(ctx, "id", uuid.New().String()[0:6])

	ctxServices, cancelServices := context.WithCancel(ctxNew)

	l := &Limit{
		Mutex:             &sync.Mutex{},
		Activities:        make(map[string]map[string]*models.Activity),
		Sessions:          make(map[string]map[string]int64),
		ActivityBySession: make(map[string]*models.Activity),
		UserBySession:     make(map[string]string),
		Max:               2,
		FlushInterval:     time.Hour,
		ProvisionalPeriod: time.Hour,
		register:          make(chan confirmation),
		Locked:            false,
		ctx:               ctx,
		ctxServices:       ctxServices,
		cancelServices:    cancelServices,
		LastBookingEnds:   time.Now().Unix(),
		Now:               func() int64 { return time.Now().Unix() },
	}

	go l.handleRegister(ctxServices)
	return l
}

func (l *Limit) ExportAll() ([]byte, error) {
	return json.Marshal(l)
}

// ImportAll takes the booking store marshalled into b as our new bookingstore
// manage pointer scope by making this return the pointer
// for the calling function to replace the poolstore pointer with
func ImportAll(l *Limit, b []byte) (*Limit, error) {

	new := &Limit{}

	err := json.Unmarshal(b, new)

	if err != nil {
		log.WithFields(log.Fields{"source": "bookingstore", "event": "import:marshal:error", "error": err.Error()}).Trace("bookingstore:import:marshal:error")
		return nil, err
	}

	log.WithFields(log.Fields{"source": "bookingstore", "event": "import:marshal:ok"}).Trace("bookingstore:import:marshal:ok")
	// pass in the parent context which we
	// are keeping
	new.PostImportEssential(l.ctx)

	// stop the old services - including any pending bookings.
	l.cancelServices()

	return new, nil
}

// PostImportEssential sets up mutexes and Now() functions
// Assume the original bookingstore context was cancelled
// so as to stop the registerhandler and flush....
func (l *Limit) PostImportEssential(ctx context.Context) {
	l.Mutex = &sync.Mutex{}
	l.register = make(chan confirmation)
	ctxNew := context.WithValue(ctx, "id", uuid.New().String()[0:6])
	ctxServices, cancelServices := context.WithCancel(ctxNew)
	l.ctxServices = ctxServices
	l.cancelServices = cancelServices
	go l.handleRegister(ctxServices)
}

// handleRegister handles booking confirmations and should be run in one separate goro
// it must also have its own context so we can stop and start it for import.
func (l *Limit) handleRegister(ctx context.Context) {

	log.WithFields(log.Fields{"source": "bookingstore", "event": "confirm:start", "ctx": ctx.Value("id")}).Trace("bookstore:confirm:start")
	for {
		select {
		case <-ctx.Done():
			log.WithFields(log.Fields{"source": "bookingstore", "event": "confirm:stop", "ctx": ctx.Value("id")}).Trace("bookingstore:confirm:stop")
			return
		case c := <-l.register:

			if c.expiresAt > l.LastBookingEnds {
				l.LastBookingEnds = c.expiresAt
			}

			l.UserBySession[c.sessionID] = c.userID

			if c.activity == nil {
				lf := log.Fields{
					"source":     "bookingstore",
					"event":      "confirm:activity:nil",
					"userID":     c.userID,
					"sessionID":  c.sessionID,
					"expiresAt":  c.expiresAt,
					"activityID": "",
					"ctx":        ctx.Value("id"),
				}
				log.WithFields(lf).Trace("bookingstore:confirm:activity:nil")
				continue
			}

			if _, ok := l.Activities[c.userID]; !ok {
				l.Activities[c.userID] = make(map[string]*models.Activity)
			}
			lau := l.Activities[c.userID]
			lau[c.sessionID] = c.activity
			l.Activities[c.userID] = lau
			ID := ""

			if c.activity.Description != nil {
				ID = c.activity.Description.ID
			}

			l.ActivityBySession[c.sessionID] = c.activity

			lf := log.Fields{
				"source":     "bookingstore",
				"event":      "confirm:activity:nil",
				"userID":     c.userID,
				"sessionID":  c.sessionID,
				"expiresAt":  c.expiresAt,
				"activityID": ID,
				"ctx":        ctx.Value("id"),
			}
			log.WithFields(lf).Trace("bookingstore:confirm:activity:ok")
		}
	}

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

	l.FlushInterval = interval

	go func() {

		lf := log.Fields{
			"source": "bookingstore",
			"event":  "withFlush:start",
		}
		log.WithFields(lf).Trace("bookingstore:withFlush:start")

		for {
			select {
			case <-l.ctxServices.Done():
				lf := log.Fields{
					"source": "bookingstore",
					"event":  "withFlush:stop",
				}
				log.WithFields(lf).Trace("bookingstore:withFlush:stop")
				return
			case <-time.After(l.FlushInterval):
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

	l.ProvisionalPeriod = interval

	return l
}

// WithMax sets the maximum number of concurrent sessions a user can have
func (l *Limit) WithMax(max int) *Limit {
	l.Lock()
	defer l.Unlock()
	l.Max = max
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

func (l *Limit) SetNow(now func() int64) {
	l.Lock()
	defer l.Unlock()
	l.Now = now
}

func (l *Limit) LockBookings() {
	l.Lock()
	defer l.Unlock()

	l.Locked = true
}

func (l *Limit) UnlockBookings() {
	l.Lock()
	defer l.Unlock()

	l.Locked = false
}

func (l *Limit) GetLockBookings() bool {
	l.Lock()
	defer l.Unlock()

	return l.Locked
}

// GetUserActivities provides pointers to all of a users activities, so that
// they can be provided with their current bookings (intended for both user and admin scope)
func (l *Limit) GetUserActivities(user string) (map[string]*models.Activity, error) {
	l.Lock()
	defer l.Unlock()

	var err error

	a, ok := l.Activities[user]
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

	return l.ActivityBySession
}

func (l *Limit) GetAllActivitiesCount() int {
	l.Lock()
	defer l.Unlock()

	return len(l.ActivityBySession)
}

// GetLastBookingEnds is an admin function to help figure out
// how long until the system is free of booked sessions
// e.g. after disabling new bookings, existing bookings
// will still have some time to run
func (l *Limit) GetLastBookingEnds() int64 {
	l.Lock()
	defer l.Unlock()

	return l.LastBookingEnds
}

// Get all sessionCount is primarily an admin function
// for a bit of dashboard eye-candy, but there is no reason
// that users can't see this too
func (l *Limit) GetAllSessionCount() int {
	l.FlushAll()
	l.Lock()
	defer l.Unlock()
	return len(l.ActivityBySession)
}

// GetActivityFromSessionID allows an activity to be retrieved even if the
// userID has been lost -> admin scope only!
func (l *Limit) GetActivityFromSessionID(sid string) (*models.Activity, error) {
	l.FlushAll()
	l.Lock()
	defer l.Unlock()

	var err error

	a, ok := l.ActivityBySession[sid]

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
	return len(l.Sessions[user])
}

// FlushAll removes all stale entries from all maps
func (l *Limit) FlushAll() {
	l.Lock()
	defer l.Unlock()

	timeSinceLastFlush := time.Now().Unix() - l.LastFlush

	if timeSinceLastFlush < 1 { // nothing has changed within the last second.
		return
	}

	sessions := l.Sessions
	for userID, _ := range sessions {
		l.flushUser(userID)
	}

	l.LastFlush = time.Now().Unix()
}

// flushUser removes stale entries from all maps
// for a given user
func (l *Limit) flushUser(user string) {

	stale := l.Sessions[user]
	now := l.Now()
	for k, s := range stale {
		if s < now {
			delete(l.Sessions[user], k)
			delete(l.Activities[user], k)
			delete(l.ActivityBySession, k)
			delete(l.UserBySession, k)
		}
	}

	removed := len(stale) - len(l.Sessions[user])

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
func (l *Limit) autoDelete(cancel, confirm chan struct{}, userID, sessionID string, ctx context.Context) {

	cancelFunc := func() {

		l.Lock()
		defer l.Unlock()

		lf := log.Fields{
			"source":    "bookingstore",
			"event":     "request:provisional:autoDelete:deleteBooking",
			"who":       userID,
			"sessionID": sessionID,
			"ctx":       ctx.Value("id"),
		}

		sessions := l.Sessions

		s, ok := sessions[userID]

		if !ok {
			log.WithFields(lf).Trace("bookingstore:request:provisional:autoDelete:deleteBooking:error:noSession")
			return
		}

		delete(s, sessionID)

		sessions[userID] = s

		l.Sessions = sessions

		log.WithFields(lf).Trace("bookingstore:request:provisional:autoDelete:deleteBooking:ok")
	}

	select {

	case <-cancel:
		cancelFunc()
		return

	case <-time.After(l.ProvisionalPeriod):
		// prevent leakage if api handler stalls before
		// confirming or cancelling
		cancelFunc()
		return

	case <-ctx.Done():
		// done because server shut down
		lf := log.Fields{
			"source":    "bookingstore",
			"event":     "request:provisional:autoDelete:contextCancelledBeforeDecision",
			"who":       userID,
			"sessionID": sessionID,
			"ctx":       ctx.Value("id"),
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
			"ctx":       ctx.Value("id"),
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
	if l.Max < 1 || l.Locked {
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
	_, ok := l.Sessions[userID]

	if !ok {
		l.Sessions[userID] = make(map[string]int64)
	} else {
		l.flushUser(userID) //up date our session list
	}

	// user session count
	usc := len(l.Sessions[userID])

	if usc >= l.Max {

		lf := log.Fields{
			"source":           "bookingstore",
			"event":            "request:provisional:denied:overLimit",
			"userID":           userID,
			"exp":              exp,
			"sessionID":        sessionID,
			"userSessionCount": usc,
			"max":              l.Max,
		}
		log.WithFields(lf).Debug("bookingstore:request:provisional:denied:overLimit")
		return nil, nil, "", errors.New("denied: over limit")
	}

	// grant provisional session

	l.Sessions[userID][sessionID] = exp

	lf := log.Fields{
		"source":           "bookingstore",
		"event":            "request:provisional:granted",
		"userID":           userID,
		"exp":              exp,
		"sessionID":        sessionID,
		"userSessionCount": usc,
		"max":              l.Max,
	}
	log.WithFields(lf).Debug("bookingstore:request:provisional:granted")

	go l.autoDelete(cancel, confirm, userID, sessionID, l.ctxServices)
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
