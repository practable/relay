package deny

import (
	"sync"
	"time"
)

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

func New() *Store {
	return &Store{
		sync.Mutex{},
		make(map[string]int64),
		make(chan struct{}),
		make(map[string]int64),
		SystemNow,
	}
}

func (s *Store) SetNowFunc(nf func() int64) {
	s.Now = nf
}

func SystemNow() int64 {
	return time.Now().Unix()
}

func (s *Store) Allow(ID string, expiresAt int64) {
	s.Lock()
	defer s.Unlock()

	//remove from Deny list, if already there
	if _, ok := s.DenyList[ID]; ok {
		delete(s.DenyList, ID)
	}
	s.AllowList[ID] = expiresAt
}

func (s *Store) Deny(ID string, expiresAt int64) {
	s.Lock()
	defer s.Unlock()

	//remove from Allow list, if already there
	if _, ok := s.AllowList[ID]; ok {
		delete(s.AllowList, ID)
	}
	s.DenyList[ID] = expiresAt
}

func (s *Store) IsDenied(ID string) bool {
	s.Lock()
	defer s.Unlock()

	_, ok := s.DenyList[ID]

	return ok
}

func (s *Store) GetDenyList() []string {
	s.Lock()
	defer s.Unlock()
	d := []string{}
	for k, _ := range s.DenyList {
		d = append(d, k)
	}
	return d
}

func (s *Store) GetAllowList() []string {
	s.Lock()
	defer s.Unlock()
	a := []string{}
	for k, _ := range s.AllowList {
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

// Prune removes stale entries from the Allow, Deny lists
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
