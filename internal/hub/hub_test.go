package hub

import (
	"bytes"
	"crypto/rand"
	"reflect"
	"testing"
	"time"

	"github.com/practable/relay/internal/counter"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {

	log.SetLevel(log.WarnLevel)

}

func TestInstantiateHub(t *testing.T) {

	h := New()

	if reflect.TypeOf(h.Broadcast) != reflect.TypeOf(make(chan Message)) {
		t.Error("Hub.Broadcast channel of wrong type")
	}
	if reflect.TypeOf(h.Register) != reflect.TypeOf(make(chan *Client)) {
		t.Error("Hub.Register channel of wrong type")
	}
	if reflect.TypeOf(h.Unregister) != reflect.TypeOf(make(chan *Client)) {
		t.Error("Hub.Unregister channel of wrong type")
	}

	if reflect.TypeOf(h.Clients) != reflect.TypeOf(make(map[string]map[*Client]bool)) {
		t.Error("Hub.Broadcast channel of wrong type")
	}

}

func TestRegisterClient(t *testing.T) {

	topic := "/video0"
	h := New()
	closed := make(chan struct{})
	go h.Run(closed)
	c := &Client{Hub: h, Name: "aa", Topic: topic, Send: make(chan Message)}

	h.Register <- c

	if val, ok := h.Clients[topic][c]; !ok {
		t.Error("Client not registered in topic")
	} else if val == false {
		t.Error("Client registered but not made true in map")
	}
	close(closed)
}

func TestUnRegisterClient(t *testing.T) {

	h := New()
	closed := make(chan struct{})

	go h.Run(closed)

	topic := "/video0"

	c := &Client{Hub: h, Name: "aa", Topic: topic, Send: make(chan Message)}

	h.Register <- c

	if val, ok := h.Clients[topic][c]; !ok {
		t.Error("Client not registered in topic")
	} else if val == false {
		t.Error("Client registered but not made true in map")
	}

	time.Sleep(time.Millisecond)
	h.Unregister <- c
	time.Sleep(time.Millisecond)
	if val, ok := h.Clients[topic][c]; ok {
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
	c1 := &Client{Hub: h, Name: "1", Topic: topicA, Send: make(chan Message)}
	c2 := &Client{Hub: h, Name: "2", Topic: topicA, Send: make(chan Message)}

	topicB := "/videoB"
	c3 := &Client{Hub: h, Name: "2", Topic: topicB, Send: make(chan Message)}

	h.Register <- c1
	h.Register <- c2
	h.Register <- c3

	content := []byte{'t', 'e', 's', 't'}

	m := &Message{Data: content, Sender: *c1, Sent: time.Now(), Type: 0}

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

func TestSendLargeMessage(t *testing.T) {

	h := New()
	closed := make(chan struct{})
	go h.Run(closed)

	topicA := "/videoA"
	c1 := &Client{Hub: h, Name: "1", Topic: topicA, Send: make(chan Message)}
	c2 := &Client{Hub: h, Name: "2", Topic: topicA, Send: make(chan Message)}

	topicB := "/videoB"
	c3 := &Client{Hub: h, Name: "2", Topic: topicB, Send: make(chan Message)}

	h.Register <- c1
	h.Register <- c2
	h.Register <- c3

	content := make([]byte, 1024)

	n, err := rand.Read(content)
	assert.NoError(t, err)
	assert.Equal(t, len(content), n)

	m := &Message{Data: content, Sender: *c1, Sent: time.Now(), Type: 0}

	var start time.Time

	go func() {
		start = time.Now()
		h.Broadcast <- *m
		time.Sleep(time.Second)
	}()

	timer := time.NewTimer(time.Millisecond)

	rxCount := counter.New()

COLLECT:
	for {
		select {
		case <-c1.Send:
			t.Error("Sender received echo")
		case msg := <-c2.Send:
			elapsed := time.Since(start)
			if elapsed > (time.Millisecond) {
				t.Error("Message took longer than 1 millisecond, size was ", len(msg.Data))
			}
			rxCount.Increment()
			if !bytes.Equal(msg.Data, content) {
				t.Error("Wrong data in message")
			}
		case <-c3.Send:
			t.Error("Wrong client received message")
		case <-timer.C:
			break COLLECT
		}
	}

	if rxCount.Read() != 1 {
		t.Error("Receiver did not receive message in correct quantity, wanted 1 got ", rxCount.Read())
	}
	close(closed)
}

func TestSendManyMessages(t *testing.T) {

	h := New()
	closed := make(chan struct{})
	go h.Run(closed)

	topicA := "/videoA"

	ca1 := &Client{Hub: h, Name: "a1", Topic: topicA, Send: make(chan Message)}
	ca2 := &Client{Hub: h, Name: "a2", Topic: topicA, Send: make(chan Message)}
	ca3 := &Client{Hub: h, Name: "a3", Topic: topicA, Send: make(chan Message)}

	topicB := "/videoB"

	cb1 := &Client{Hub: h, Name: "b1", Topic: topicB, Send: make(chan Message)}
	cb2 := &Client{Hub: h, Name: "b2", Topic: topicB, Send: make(chan Message)}
	cb3 := &Client{Hub: h, Name: "b3", Topic: topicB, Send: make(chan Message)}

	topicC := "/videoC"

	cc1 := &Client{Hub: h, Name: "c1", Topic: topicC, Send: make(chan Message)}
	cc2 := &Client{Hub: h, Name: "c2", Topic: topicC, Send: make(chan Message)}
	cc3 := &Client{Hub: h, Name: "c3", Topic: topicC, Send: make(chan Message)}

	topicD := "/videoD"

	cd1 := &Client{Hub: h, Name: "d1", Topic: topicD, Send: make(chan Message)}
	cd2 := &Client{Hub: h, Name: "d2", Topic: topicD, Send: make(chan Message)}
	cd3 := &Client{Hub: h, Name: "d3", Topic: topicD, Send: make(chan Message)}

	h.Register <- ca1
	h.Register <- ca2
	h.Register <- ca3
	h.Register <- cb1
	h.Register <- cb2
	h.Register <- cb3
	h.Register <- cc1
	h.Register <- cc2
	h.Register <- cc3
	h.Register <- cd1
	h.Register <- cd2
	h.Register <- cd3

	time.Sleep(time.Millisecond)

	if len(h.Clients[topicA]) != 3 {
		t.Error("Wrong number of clients registered for TopicA, wanted 3 got", len(h.Clients[topicA]))
	}
	if len(h.Clients[topicB]) != 3 {
		t.Error("Wrong number of clients registered for TopicB, wanted 3 got", len(h.Clients[topicB]))
	}
	if len(h.Clients[topicC]) != 3 {
		t.Error("Wrong number of clients registered for TopicC, wanted 3 got", len(h.Clients[topicB]))
	}
	if len(h.Clients[topicD]) != 3 {
		t.Error("Wrong number of clients registered for TopicD, wanted 3 got", len(h.Clients[topicB]))
	}
	contentA := make([]byte, 1024*1024*10)
	contentB := make([]byte, 1024*1024*10)
	contentC := make([]byte, 1024*1024*10)
	contentD := make([]byte, 1024*1024*10)

	n, err := rand.Read(contentA)
	assert.NoError(t, err)
	assert.Equal(t, len(contentA), n)

	n, err = rand.Read(contentB)
	assert.NoError(t, err)
	assert.Equal(t, len(contentB), n)

	n, err = rand.Read(contentC)
	assert.NoError(t, err)
	assert.Equal(t, len(contentC), n)

	n, err = rand.Read(contentD)
	assert.NoError(t, err)
	assert.Equal(t, len(contentD), n)

	mA := &Message{Data: contentA, Sender: *ca1, Sent: time.Now(), Type: 0}
	mB := &Message{Data: contentB, Sender: *cb1, Sent: time.Now(), Type: 0}
	mC := &Message{Data: contentC, Sender: *cc1, Sent: time.Now(), Type: 0}
	mD := &Message{Data: contentD, Sender: *cd1, Sent: time.Now(), Type: 0}

	rxCount := counter.New()

	iterations := 100

	duration := time.Duration(iterations+5)*5*time.Millisecond + 20*time.Millisecond

	go receive(rxCount, ca2, contentA, duration, t)
	go receive(rxCount, cb2, contentB, duration, t)
	go receive(rxCount, cc2, contentC, duration, t)
	go receive(rxCount, cd2, contentD, duration, t)
	go receive(rxCount, ca3, contentA, duration, t)
	go receive(rxCount, cb3, contentB, duration, t)
	go receive(rxCount, cc3, contentC, duration, t)
	go receive(rxCount, cd3, contentD, duration, t)

	time.Sleep(time.Millisecond)

	for i := 0; i < iterations; i++ {
		h.Broadcast <- *mA
		h.Broadcast <- *mB
		h.Broadcast <- *mC
		h.Broadcast <- *mD
		time.Sleep(5 * time.Millisecond)
	}

	time.Sleep(duration)

	if rxCount.Read() != iterations*8 {
		t.Error("Got wrong message count, wanted/got", iterations*8, rxCount.Read())
	}

	close(closed)

}

func receive(counter *counter.Counter, client *Client, content []byte, duration time.Duration, t *testing.T) {

	timer := time.NewTimer(duration)

COLLECT:
	for {
		select {
		case msg := <-client.Send:
			if !bytes.Equal(msg.Data, content) {
				t.Error("Wrong data in message", len(msg.Data), len(content))
			} else {
				counter.Increment()
			}
		case <-timer.C:
			break COLLECT
		}
	}
}
