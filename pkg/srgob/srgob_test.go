package srgob

import (
	"bytes"
	"encoding/gob"
	"testing"
)

func TestEncodeDecode(t *testing.T) {

	var buf bytes.Buffer

	m1 := Message{
		ID:   "5678-efgh",
		Data: []byte("This is a test message\nIsn't it"),
	}

	expectedString := "5678-efgh 31"

	encoder := gob.NewEncoder(&buf)

	err := encoder.Encode(m1)

	var m2 Message

	decoder := gob.NewDecoder(&buf)

	err = decoder.Decode(&m2)

	if err != nil {
		t.Errorf("Fatal error %v", err.Error())
	}

	if m2.String() != expectedString {
		t.Errorf("Didn't get expected string.\nWanted: %s\nGot   : %s\n", expectedString, m2.String())
	}

}
