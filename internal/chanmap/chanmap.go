package chanmap

import (
	"errors"
	"sync"
)

// Store maps parents and children channels e.g. for associating with bookingIDs
type Store struct {
	*sync.Mutex

	// ChildrenByParent holds a map of child channels
	ChildrenByParent map[string]map[string]chan struct{}
	// ParentByChild helps us delete efficiently, by telling us which parent map the child is in
	ParentByChild map[string]string
}

// New returns a store with initialised maps
func New() *Store {

	return &Store{
		&sync.Mutex{},
		make(map[string]map[string]chan struct{}),
		make(map[string]string),
	}

}

// Add a parent-child relationship. A parent is typically a bookingID, child for clientname
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

	if _, ok := s.ChildrenByParent[parent]; !ok {
		s.ChildrenByParent[parent] = make(map[string]chan struct{})
	}

	p := s.ChildrenByParent[parent]

	p[child] = ch

	s.ChildrenByParent[parent] = p

	s.ParentByChild[child] = parent

	return nil
}

// DeleteChild deletes the child, without closing the channel
func (s *Store) DeleteChild(child string) error {
	s.Lock()
	defer s.Unlock()

	return s.deleteAndOptionalCloseChild(child, false)

}

// DeleteAndCloseChild closes the child's channel and deletes it
// this approach ensures the channel cannot be closed twice
func (s *Store) DeleteAndCloseChild(child string) error {
	s.Lock()
	defer s.Unlock()

	return s.deleteAndOptionalCloseChild(child, true)

}

// deleteAndOptionalCloseChild is for internal use only by functions holding the lock already
func (s *Store) deleteAndOptionalCloseChild(child string, closeChannel bool) error {

	if child == "" {
		return errors.New("no child")
	}

	// if child not in map, no error - already gone
	if parent, ok := s.ParentByChild[child]; ok {

		children := s.ChildrenByParent[parent]

		if ch, ok := children[child]; ok {

			if closeChannel {
				close(ch)
			}
			delete(children, child)
		}

		s.ChildrenByParent[parent] = children

		delete(s.ParentByChild, child)
	}

	return nil

}

// DeleteParent deletes the parent, and all its children, without closing the children's channels
func (s *Store) DeleteParent(parent string) error {
	s.Lock()
	defer s.Unlock()

	return s.deleteAndOptionalCloseParent(parent, false)

}

// DeleteAndCloseParent deletes the parent, and all its children, closing the children's channels
func (s *Store) DeleteAndCloseParent(parent string) error {
	s.Lock()
	defer s.Unlock()

	return s.deleteAndOptionalCloseParent(parent, true)

}

// deleteAndOptionalCloseParent is for internal use only by functions holding the lock already
func (s *Store) deleteAndOptionalCloseParent(parent string, closeChannel bool) error {

	if parent == "" {
		return errors.New("no parent")
	}

	// if parent not in map, no error - already gone
	if children, ok := s.ChildrenByParent[parent]; ok {

		for _, ch := range children {
			if closeChannel {
				close(ch)
			}
		}

		delete(s.ChildrenByParent, parent)
	}

	return nil

}
