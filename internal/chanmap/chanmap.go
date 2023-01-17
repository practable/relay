package chanmap

import (
	"errors"
	"sync"
)

type Store struct {
	*sync.Mutex

	// ChildByParent holds a map of child channels
	ChildByParent map[string]map[string]chan struct{}
	// ParentByChild helps us delete efficiently, by telling us which parent map the child is in
	ParentByChild map[string]string
}

func New() *Store {

	return &Store{
		&sync.Mutex{},
		make(map[string]map[string]chan struct{}),
		make(map[string]string),
	}

}

// parent is typically a bookingID, child for clientname
func (s *Store) Add(parent, child string, ch chan struct{}) error {

	s.Lock()
	defer s.Unlock()

	if parent == "" {
		return errors.New("no parent")
	}
	if child == "" {
		return errors.New("no child")
	}
	if ch == nil {
		return errors.New("no channel")
	}

	if _, ok := s.ChildByParent[parent]; !ok {
		s.ChildByParent[parent] = make(map[string]chan struct{})
	}

	p := s.ChildByParent[parent]

	p[child] = ch

	s.ChildByParent[parent] = p

	s.ParentByChild[child] = parent

	return nil
}

// Delete deletes the child, without closing the channel
func (s *Store) Delete(child string) error {
	s.Lock()
	defer s.Unlock()

	return s.deleteAndOptionalClose(child, false)

}

// DeleteAndClose closes the child's channel and deletes it
// this approach ensures the channel cannot be closed twice
func (s *Store) DeleteAndClose(child string) error {
	s.Lock()
	defer s.Unlock()

	return s.deleteAndOptionalClose(child, true)

}

// deleteAndOptionalClose is for internal use only by functions holding the lock already
func (s *Store) deleteAndOptionalClose(child string, closeChannel bool) error {

	if child == "" {
		return errors.New("must have a non-zero length string for child")
	}

	// if child not in map, no error - already gone
	if parent, ok := s.ParentByChild[child]; ok {

		p := s.ChildByParent[parent]

		if ch, ok := p[child]; ok {

			if closeChannel {
				close(ch)
			}
			delete(p, child)
		}

		s.ChildByParent[parent] = p

		delete(s.ParentByChild, child)
	}

	return nil

}
