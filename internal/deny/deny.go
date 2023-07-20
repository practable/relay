package deny

import (
	"sync"
	"time"
)

// Store tracks current connections, and those that have been denied (cancelled)
type Store struct {
	sync.Mutex

	// map of IDs currently connected, with expiry time
	AllowList map[string]int64

	// for shutting down the keepclean routine
	closed chan struct{}

	// bookingIDs currently denied, with expiry time
	DenyList map[string]int64

	// Now is a function for getting the time - useful for mocking in test
	// note time is in int64 format
	Now func() int64 `json:"-" yaml:"-"`
}

// New returns a store with the maps initialised
func New() *Store {
	return &Store{
		sync.Mutex{},
		make(map[string]int64),
		make(chan struct{}),
		make(map[string]int64),
		SystemNow,
	}
}

// SetNowFunc allows providing an alternative source of the current time, for testing purposes
func (s *Store) SetNowFunc(nf func() int64) {
	s.Now = nf
}

// SystemNow returns the current system time
func SystemNow() int64 {
	return time.Now().Unix()
}

// Allow reverts a denied ID back to being allowed
func (s *Store) Allow(ID string, expiresAt int64) {
	s.Lock()
	defer s.Unlock()

	//remove from Deny list, if already there
	if _, ok := s.DenyList[ID]; ok {
		delete(s.DenyList, ID)
	}
	s.AllowList[ID] = expiresAt
}

// Deny adds and ID to the deny list
func (s *Store) Deny(ID string, expiresAt int64) {
	s.Lock()
	defer s.Unlock()

	//remove from Allow list, if already there
	if _, ok := s.AllowList[ID]; ok {
		delete(s.AllowList, ID)
	}
	s.DenyList[ID] = expiresAt
}

// IsDenied checks if an ID is on the denied list
func (s *Store) IsDenied(ID string) bool {
	s.Lock()
	defer s.Unlock()

	_, ok := s.DenyList[ID]

	return ok
}

// GetDenyList returns the entire deny list
func (s *Store) GetDenyList() []string {
	s.Lock()
	defer s.Unlock()
	d := []string{}
	for k := range s.DenyList {
		d = append(d, k)
	}
	return d
}

// GetAllowList returns the entire allow list
func (s *Store) GetAllowList() []string {
	s.Lock()
	defer s.Unlock()
	a := []string{}
	for k := range s.AllowList {
		a = append(a, k)
	}
	return a
}

// Prune removes stale entries from the Allow, Deny lists
func (s *Store) Prune() {
	s.Lock()
	defer s.Unlock()

	s.prune()
}

// prune removes stale entries from the Allow, Deny lists
// internal usage only as does not take the lock
func (s *Store) prune() {

	now := s.Now()

	stale := []string{}

	for k, v := range s.AllowList {
		if v < now {
			stale = append(stale, k)
		}
	}

	for _, ID := range stale {
		delete(s.AllowList, ID)
	}

	stale = []string{}

	for k, v := range s.DenyList {
		if v < now {
			stale = append(stale, k)
		}
	}

	for _, ID := range stale {
		delete(s.DenyList, ID)
	}
}
