package agg

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/timdrysdale/relay/pkg/hub"
)

func TestInstantiateHub(t *testing.T) {

	h := New()

	if reflect.TypeOf(h.Broadcast) != reflect.TypeOf(make(chan hub.Message)) {
		t.Error("Hub.Broadcast channel of wrong type")
	}
	if reflect.TypeOf(h.Register) != reflect.TypeOf(make(chan *hub.Client)) {
		t.Error("Hub.Register channel of wrong type")
	}
	if reflect.TypeOf(h.Unregister) != reflect.TypeOf(make(chan *hub.Client)) {
		t.Error("Hub.Unregister channel of wrong type")
	}

	if reflect.TypeOf(h.Streams) != reflect.TypeOf(make(map[string]map[*hub.Client]bool)) {
		t.Error("Hub.Broadcast channel of wrong type")
	}

	if reflect.TypeOf(h.Rules) != reflect.TypeOf(make(map[string][]string)) {
		t.Error("Hub.Broadcast channel of wrong type")
	}

}

func TestRegisterClient(t *testing.T) {
	// also exercises the RunWithStats path
	for i := 0; i < 2; i++ {
		topic := "/video0"
		h := New()
		closed := make(chan struct{})

		if i == 0 {
			go h.Run(closed)
		} else {
			go h.RunWithStats(closed)
		}

		c := &hub.Client{Hub: h.Hub, Name: "aa", Topic: topic, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

		h.Register <- c

		time.Sleep(time.Millisecond)

		if val, ok := h.Hub.Clients[topic][c]; !ok {
			t.Error("Client not registered in topic")
		} else if val == false {
			t.Error("Client registered but not made true in map")
		}
		close(closed)
	}
}

func TestUnRegisterClient(t *testing.T) {

	topic := "/video0"
	h := New()
	closed := make(chan struct{})
	go h.Run(closed)
	c := &hub.Client{Hub: h.Hub, Name: "aa", Topic: topic, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	h.Register <- c

	time.Sleep(time.Millisecond)

	if val, ok := h.Hub.Clients[topic][c]; !ok {
		t.Error("Client not registered in topic")
	} else if val == false {
		t.Error("Client registered but not made true in map")
	}

	time.Sleep(time.Millisecond)
	h.Unregister <- c
	time.Sleep(time.Millisecond)
	if val, ok := h.Hub.Clients[topic][c]; ok {
		if val {
			t.Error("Client still registered")
		}
	}
	close(closed)
}

func TestSendMessage(t *testing.T) {

	h := New()
	closed := make(chan struct{})
	go h.Run(closed)

	topicA := "/videoA"
	c1 := &hub.Client{Hub: h.Hub, Name: "1", Topic: topicA, Send: make(chan hub.Message), Stats: hub.NewClientStats()}
	c2 := &hub.Client{Hub: h.Hub, Name: "2", Topic: topicA, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	topicB := "/videoB"
	c3 := &hub.Client{Hub: h.Hub, Name: "2", Topic: topicB, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	h.Register <- c1
	h.Register <- c2
	h.Register <- c3

	content := []byte{'t', 'e', 's', 't'}

	m := &hub.Message{Data: content, Sender: *c1, Sent: time.Now(), Type: 0}

	var start time.Time

	rxCount := 0

	go func() {
		timer := time.NewTimer(5 * time.Millisecond)
	COLLECT:
		for {
			select {
			case <-c1.Send:
				t.Error("Sender received echo")
			case msg := <-c2.Send:
				elapsed := time.Since(start)
				if elapsed > (time.Millisecond) {
					t.Error("Message took longer than 1 millisecond, ", elapsed)
				}
				rxCount++
				if !bytes.Equal(msg.Data, content) {
					t.Error("Wrong data in message")
				}
			case <-c3.Send:
				t.Error("Wrong client received message")
			case <-timer.C:
				break COLLECT
			}
		}
	}()

	time.Sleep(time.Millisecond)
	start = time.Now()
	h.Broadcast <- *m
	time.Sleep(time.Millisecond)
	if rxCount != 1 {
		t.Error("Receiver did not receive message in correct quantity, wanted 1 got ", rxCount)
	}
	close(closed)
}

func TestRegisterStreamNoRule(t *testing.T) {

	topic := "stream/video0"
	h := New()
	closed := make(chan struct{})
	defer close(closed)

	go h.Run(closed)
	c := &hub.Client{Hub: h.Hub, Name: "aa", Topic: topic, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	h.Register <- c

	time.Sleep(time.Millisecond)

	if val, ok := h.Streams[topic][c]; !ok {
		t.Error("Stream not registered in topic")
	} else if val == false {
		t.Error("Stream registered but not made true in map")
	}

}

func TestUnRegisterStreamNoRule(t *testing.T) {

	topic := "stream/video0"
	h := New()
	closed := make(chan struct{})
	defer close(closed)

	go h.Run(closed)
	c := &hub.Client{Hub: h.Hub, Name: "aa", Topic: topic, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	h.Register <- c

	time.Sleep(time.Millisecond)

	if val, ok := h.Streams[topic][c]; !ok {
		t.Error("Stream not registered in topic")
	} else if val == false {
		t.Error("Client registered but not made true in map")
	}

	time.Sleep(time.Millisecond)
	h.Unregister <- c
	time.Sleep(time.Millisecond)
	if val, ok := h.Streams[topic][c]; ok {
		if val {
			t.Error("Stream still registered")
		}
	}

}

func TestAddRuleNoStream(t *testing.T) {
	h := New()
	closed := make(chan struct{})
	defer close(closed)

	go h.Run(closed)

	stream := "stream/large"
	feeds := []string{"video0", "audio"}
	r := &Rule{Stream: stream, Feeds: feeds}

	h.Add <- *r

	time.Sleep(time.Millisecond)

	if val, ok := h.Rules[stream]; !ok {
		t.Error("Rule not registered in Rules")

	} else if len(val) != len(feeds) {
		t.Error("Rule has incorrect number of feeds")
	}

}

func TestCannotAddRuleDeleteAll(t *testing.T) {
	h := New()
	closed := make(chan struct{})
	defer close(closed)
	go h.Run(closed)

	stream := "deleteAll"
	feeds := []string{"video0", "audio"}
	r := &Rule{Stream: stream, Feeds: feeds}

	h.Add <- *r

	time.Sleep(time.Millisecond)

	if _, ok := h.Rules[stream]; ok {
		t.Error("Rule called deleteAll incorrectly accepted for registering in Rules")
	}
}

func TestDeleteRuleNoStream(t *testing.T) {
	h := New()
	closed := make(chan struct{})
	go h.Run(closed)
	defer close(closed)
	stream := "stream/large"
	feeds := []string{"video0", "audio"}
	r := &Rule{Stream: stream, Feeds: feeds}

	h.Add <- *r

	time.Sleep(time.Millisecond)

	if val, ok := h.Rules[stream]; !ok {
		t.Error("Rule not registered in Rules")

	} else if len(val) != len(feeds) {
		t.Error("Rule has incorrect number of feeds")
	}

	h.Delete <- (*r).Stream

	time.Sleep(time.Millisecond)

	if _, ok := h.Rules[stream]; ok {
		t.Error("Rule still registered in Rules")

	}

}

func TestDeleteAllRules(t *testing.T) {
	h := New()
	closed := make(chan struct{})
	defer close(closed)
	go h.Run(closed)

	stream0 := "stream/large"
	feeds := []string{"video0", "audio"}
	r := &Rule{Stream: stream0, Feeds: feeds}

	h.Add <- *r

	stream1 := "stream/medium"
	feeds = []string{"video1", "audio"}
	r = &Rule{Stream: stream1, Feeds: feeds}

	h.Add <- *r

	time.Sleep(time.Millisecond)

	if val, ok := h.Rules[stream0]; !ok {
		t.Error("Rule not registered in Rules")

	} else if len(val) != len(feeds) {
		t.Error("Rule has incorrect number of feeds")
	}
	// register client to stream
	c0 := &hub.Client{Hub: h.Hub, Name: "a0", Topic: stream0, Send: make(chan hub.Message), Stats: hub.NewClientStats()}
	h.Register <- c0
	c1 := &hub.Client{Hub: h.Hub, Name: "a1", Topic: stream1, Send: make(chan hub.Message), Stats: hub.NewClientStats()}
	h.Register <- c1
	time.Sleep(time.Millisecond)

	if val, ok := h.Streams[stream0][c0]; !ok {
		t.Error("Stream not registered in topic")
	} else if val == false {
		t.Error("Stream registered but not made true in map")
	}
	if val, ok := h.Streams[stream1][c1]; !ok {
		t.Error("Stream not registered in topic")
	} else if val == false {
		t.Error("Stream registered but not made true in map")
	}

	time.Sleep(time.Millisecond)

	h.Delete <- "deleteAll"

	time.Sleep(time.Millisecond)

	if _, ok := h.Rules[stream0]; ok {
		t.Error("Rule still registered in Rules")

	}
	if _, ok := h.Rules[stream1]; ok {
		t.Error("Rule still registered in Rules")

	}

}

func TestAddRuleAddDeleteStream(t *testing.T) {
	h := New()
	closed := make(chan struct{})
	defer close(closed)
	go h.Run(closed)

	// add rule
	stream := "stream/large"

	feeds := []string{"video0", "audio"}
	r := &Rule{Stream: stream, Feeds: feeds}

	h.Add <- *r

	time.Sleep(time.Millisecond)

	if val, ok := h.Rules[stream]; !ok {
		t.Error("Rule not registered in Rules")

	} else if len(val) != len(feeds) {
		t.Error("Rule has incorrect number of feeds")
	}

	// register client to stream
	c := &hub.Client{Hub: h.Hub, Name: "aa", Topic: stream, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	h.Register <- c

	time.Sleep(time.Millisecond)

	if val, ok := h.Streams[stream][c]; !ok {
		t.Error("Stream not registered in topic")
	} else if val == false {
		t.Error("Stream registered but not made true in map")
	}

	time.Sleep(time.Millisecond)

	//Check client is registered to feeds

	listOfSubClients := h.SubClients[c] //save for post unregister test

	isFound := make([]bool, len(feeds))

	for i, feed := range feeds {

		for subclient := range listOfSubClients {

			if _, ok := h.Hub.Clients[feed][subclient.Client]; ok {
				isFound[i] = true
			}
		}
	}

	for i, val := range isFound {
		if !val {
			t.Error("did not find subclient for", feeds[i])
		}
	}

	// unregister client

	h.Unregister <- c

	time.Sleep(time.Millisecond)

	isFound = make([]bool, len(feeds))

	for i, feed := range feeds {

		for subclient := range listOfSubClients {

			if _, ok := h.Hub.Clients[feed][subclient.Client]; ok {
				isFound[i] = true
			}
		}
	}

	for i, val := range isFound {
		if val {
			t.Error("after unregistering, found subclient for", feeds[i])
		}
	}
}

func TestAddRuleAddStreamDeleteRule(t *testing.T) {
	h := New()
	closed := make(chan struct{})
	defer close(closed)
	go h.Run(closed)

	// add rule
	stream := "stream/large"

	feeds := []string{"video0", "audio"}
	r := &Rule{Stream: stream, Feeds: feeds}

	h.Add <- *r

	time.Sleep(time.Millisecond)

	if val, ok := h.Rules[stream]; !ok {
		t.Error("Rule not registered in Rules")

	} else if len(val) != len(feeds) {
		t.Error("Rule has incorrect number of feeds")
	}

	// register client to stream
	c := &hub.Client{Hub: h.Hub, Name: "aa", Topic: stream, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	h.Register <- c

	time.Sleep(time.Millisecond)

	if val, ok := h.Streams[stream][c]; !ok {
		t.Error("Stream not registered in topic")
	} else if val == false {
		t.Error("Stream registered but not made true in map")
	}

	time.Sleep(time.Millisecond)

	//Check client is registered to feeds

	listOfSubClients := h.SubClients[c] //save for post unregister test

	isFound := make([]bool, len(feeds))

	for i, feed := range feeds {

		for subclient := range listOfSubClients {

			if _, ok := h.Hub.Clients[feed][subclient.Client]; ok {
				isFound[i] = true
			}
		}
	}

	for i, val := range isFound {
		if !val {
			t.Error("did not find subclient for", feeds[i])
		}
	}

	// delete Rule

	h.Delete <- (*r).Stream

	time.Sleep(time.Millisecond)

	isFound = make([]bool, len(feeds))

	for i, feed := range feeds {

		for subclient := range listOfSubClients {

			if _, ok := h.Hub.Clients[feed][subclient.Client]; ok {
				isFound[i] = true
			}
		}
	}

	for i, val := range isFound {
		if val {
			t.Error("after deleting rule, found subclient for", feeds[i])
		}
	}
}

func TestStreamGetsFeedMessges(t *testing.T) {
	h := New()
	closed := make(chan struct{})
	defer close(closed)
	go h.Run(closed)

	// add rule
	stream := "stream/large"

	feeds := []string{"video0", "audio"}
	r := &Rule{Stream: stream, Feeds: feeds}

	h.Add <- *r

	// register client to stream
	c := &hub.Client{Hub: h.Hub, Name: "aa", Topic: stream, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	h.Register <- c

	// add feeds
	topic1 := "video0"
	c1 := &hub.Client{Hub: h.Hub, Name: "1", Topic: topic1, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	topic2 := "audio"
	c2 := &hub.Client{Hub: h.Hub, Name: "2", Topic: topic2, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	topic3 := "nothing"
	c3 := &hub.Client{Hub: h.Hub, Name: "3", Topic: topic3, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	h.Register <- c1
	h.Register <- c2
	h.Register <- c3

	content := []byte{'t', 'e', 's', 't'}

	m1 := &hub.Message{Data: content, Sender: *c1, Sent: time.Now(), Type: 0}
	m2 := &hub.Message{Data: content, Sender: *c2, Sent: time.Now(), Type: 0}

	var start time.Time

	rxCount := 0

	rxFromC1 := false
	rxFromC2 := false

	go func() {
		timer := time.NewTimer(5 * time.Millisecond)
	COLLECT:
		for {
			select {
			case <-c1.Send:
				t.Error("Sender c1 received echo")
			case <-c2.Send:
				t.Error("Sender c2 received echo")
			case <-c3.Send:
				t.Error("Wrong client received message")
			case msg := <-c.Send:
				elapsed := time.Since(start)
				if elapsed > (time.Millisecond) {
					t.Error("Message took longer than 1 millisecond, ", elapsed)
				}
				rxCount++
				if !bytes.Equal(msg.Data, content) {
					t.Error("Wrong data in message")
				}
				if msg.Sender == *c1 {
					rxFromC1 = true
				}
				if msg.Sender == *c2 {
					rxFromC2 = true
				}
			case <-c3.Send:
				t.Error("Wrong client received message")
			case <-timer.C:
				break COLLECT
			}
		}
	}()

	time.Sleep(time.Millisecond)
	start = time.Now()
	h.Broadcast <- *m1
	h.Broadcast <- *m2
	time.Sleep(time.Millisecond)
	if rxCount != 2 {
		t.Error("Receiver did not receive message in correct quantity, wanted 2 got ", rxCount)
	}
	if !rxFromC1 {
		t.Error("Did not get message from c1")
	}
	if !rxFromC2 {
		t.Error("Did not get message from c2")
	}
}

func TestStreamWithRuleChange(t *testing.T) {
	h := New()
	closed := make(chan struct{})
	defer close(closed)
	go h.Run(closed)

	// add rule
	stream := "stream/large"

	feeds := []string{"video0", "audio"}
	r := &Rule{Stream: stream, Feeds: feeds}

	h.Add <- *r

	// register client to stream
	c := &hub.Client{Hub: h.Hub, Name: "aa", Topic: stream, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	h.Register <- c

	// add feeds
	topic1 := "video0"
	c1 := &hub.Client{Hub: h.Hub, Name: "1", Topic: topic1, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	topic2 := "audio"
	c2 := &hub.Client{Hub: h.Hub, Name: "2", Topic: topic2, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	topic3 := "nothing"
	c3 := &hub.Client{Hub: h.Hub, Name: "3", Topic: topic3, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	h.Register <- c1
	h.Register <- c2
	h.Register <- c3

	content := []byte{'t', 'e', 's', 't'}

	m1 := &hub.Message{Data: content, Sender: *c1, Sent: time.Now(), Type: 0}
	m2 := &hub.Message{Data: content, Sender: *c2, Sent: time.Now(), Type: 0}
	m3 := &hub.Message{Data: content, Sender: *c3, Sent: time.Now(), Type: 0}

	var start time.Time

	rxCount := 0

	rxFromC1 := false
	rxFromC2 := false
	rxFromC3 := false

	stopRx := make(chan struct{})

	go func() {
		timer := time.NewTimer(100 * time.Millisecond)
	COLLECT:
		for {
			select {
			case <-stopRx:
				break COLLECT
			case <-c1.Send:
				t.Error("Sender c1 received echo")
			case <-c2.Send:
				t.Error("Sender c2 received echo")
			case <-c3.Send:
				t.Error("Wrong client received message")
			case msg := <-c.Send:
				elapsed := time.Since(start)
				if elapsed > (10 * time.Millisecond) {
					// it's not likely to happen - but if this test consistently fails
					// then there is a serious delay somewhere!
					t.Error("Message took longer than 10 millisecond, ", elapsed+2)
				}
				rxCount++
				if !bytes.Equal(msg.Data, content) {
					t.Error("Wrong data in message", msg.Sender.Topic, len(msg.Data))
				}
				if msg.Sender == *c1 {
					rxFromC1 = true
				}
				if msg.Sender == *c2 {
					rxFromC2 = true
				}
				if msg.Sender == *c3 {
					rxFromC3 = true
				}
			case <-timer.C:
				break COLLECT
			}
		}
	}()

	time.Sleep(time.Millisecond)

	start = time.Now()
	h.Broadcast <- *m1
	h.Broadcast <- *m2
	h.Broadcast <- *m3

	time.Sleep(time.Millisecond)

	feeds = []string{"nothing"}
	r = &Rule{Stream: stream, Feeds: feeds}
	h.Add <- *r

	time.Sleep(time.Millisecond)
	time.Sleep(time.Millisecond)
	h.Broadcast <- *m1
	h.Broadcast <- *m2
	h.Broadcast <- *m3

	time.Sleep(time.Millisecond)
	close(stopRx)

	if rxCount != 3 {
		t.Error("Receiver did not receive message in correct quantity, wanted 3 got ", rxCount)
	}
	if !rxFromC1 {
		t.Error("Did not get message from c1")
	}
	if !rxFromC2 {
		t.Error("Did not get message from c2")
	}
	if !rxFromC3 {
		t.Error("Did not get message from c3")
	}
}
