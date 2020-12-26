package srgob

import "strconv"

// return info about the message in string format (probably mainly for debugging)
func (m Message) String() string {

	s := m.ID + " " + strconv.Itoa(len(m.Data))

	return s

}
