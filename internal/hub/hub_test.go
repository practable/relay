package hub

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/practable/relay/internal/counter"
	"github.com/stretchr/testify/assert"
)

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

func TestInstantiateClient(t *testing.T) {

	h := New()
	c := &Client{Hub: h, Name: "aa", Topic: "/video0", Send: make(chan Message), Stats: NewClientStats()}

	if time.Since(c.Stats.ConnectedAt) > time.Millisecond {
		t.Error("Client connectedAt time is incorrect")
	}

	if !c.Stats.Tx.Last.IsZero() {
		t.Error("Client last Tx time is not zero", c.Stats.Tx.Last)
	}

	if c.Stats.Tx.Size.Count() != 0 {
		t.Error("Client's Tx Size stats not initialised")
	}
	if c.Stats.Tx.Dt.Count() != 0 {
		t.Error("Client's Tx Dt stats not initialised")
	}

	if !c.Stats.Rx.Last.IsZero() {
		t.Error("Client last Rx time is not zero", c.Stats.Rx.Last)
	}

	if c.Stats.Rx.Size.Count() != 0 {
		t.Error("Client's Rx Size stats not initialised")
	}
	if c.Stats.Rx.Dt.Count() != 0 {
		t.Error("Client's Rx Dt stats not initialised")
	}

}

func TestRegisterClient(t *testing.T) {

	topic := "/video0"
	h := New()
	closed := make(chan struct{})
	go h.Run(closed)
	c := &Client{Hub: h, Name: "aa", Topic: topic, Send: make(chan Message), Stats: NewClientStats()}

	h.Register <- c

	if val, ok := h.Clients[topic][c]; !ok {
		t.Error("Client not registered in topic")
	} else if val == false {
		t.Error("Client registered but not made true in map")
	}
	close(closed)
}

func TestUnRegisterClient(t *testing.T) {

	for i := 0; i < 2; i++ {

		h := New()
		closed := make(chan struct{})
		if i == 0 {
			go h.Run(closed)
		} else {
			go h.RunWithStats(closed)
		}

		topic := "/video0"

		c := &Client{Hub: h, Name: "aa", Topic: topic, Send: make(chan Message), Stats: NewClientStats()}

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
}

func TestSendMessage(t *testing.T) {

	h := New()
	closed := make(chan struct{})
	go h.Run(closed)

	topicA := "/videoA"
	c1 := &Client{Hub: h, Name: "1", Topic: topicA, Send: make(chan Message), Stats: NewClientStats()}
	c2 := &Client{Hub: h, Name: "2", Topic: topicA, Send: make(chan Message), Stats: NewClientStats()}

	topicB := "/videoB"
	c3 := &Client{Hub: h, Name: "2", Topic: topicB, Send: make(chan Message), Stats: NewClientStats()}

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
	c1 := &Client{Hub: h, Name: "1", Topic: topicA, Send: make(chan Message), Stats: NewClientStats()}
	c2 := &Client{Hub: h, Name: "2", Topic: topicA, Send: make(chan Message), Stats: NewClientStats()}

	topicB := "/videoB"
	c3 := &Client{Hub: h, Name: "2", Topic: topicB, Send: make(chan Message), Stats: NewClientStats()}

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

	ca1 := &Client{Hub: h, Name: "a1", Topic: topicA, Send: make(chan Message), Stats: NewClientStats()}
	ca2 := &Client{Hub: h, Name: "a2", Topic: topicA, Send: make(chan Message), Stats: NewClientStats()}
	ca3 := &Client{Hub: h, Name: "a3", Topic: topicA, Send: make(chan Message), Stats: NewClientStats()}

	topicB := "/videoB"

	cb1 := &Client{Hub: h, Name: "b1", Topic: topicB, Send: make(chan Message), Stats: NewClientStats()}
	cb2 := &Client{Hub: h, Name: "b2", Topic: topicB, Send: make(chan Message), Stats: NewClientStats()}
	cb3 := &Client{Hub: h, Name: "b3", Topic: topicB, Send: make(chan Message), Stats: NewClientStats()}

	topicC := "/videoC"

	cc1 := &Client{Hub: h, Name: "c1", Topic: topicC, Send: make(chan Message), Stats: NewClientStats()}
	cc2 := &Client{Hub: h, Name: "c2", Topic: topicC, Send: make(chan Message), Stats: NewClientStats()}
	cc3 := &Client{Hub: h, Name: "c3", Topic: topicC, Send: make(chan Message), Stats: NewClientStats()}

	topicD := "/videoD"

	cd1 := &Client{Hub: h, Name: "d1", Topic: topicD, Send: make(chan Message), Stats: NewClientStats()}
	cd2 := &Client{Hub: h, Name: "d2", Topic: topicD, Send: make(chan Message), Stats: NewClientStats()}
	cd3 := &Client{Hub: h, Name: "d3", Topic: topicD, Send: make(chan Message), Stats: NewClientStats()}

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

func TestStats(t *testing.T) {

	h := New()
	closed := make(chan struct{})
	go h.RunWithStats(closed)

	topicA := "/videoA"

	ca1 := &Client{Hub: h, Name: "a1", Topic: topicA, Send: make(chan Message), Stats: NewClientStats()}
	ca2 := &Client{Hub: h, Name: "a2", Topic: topicA, Send: make(chan Message), Stats: NewClientStats()}
	ca3 := &Client{Hub: h, Name: "a3", Topic: topicA, Send: make(chan Message), Stats: NewClientStats()}

	topicB := "/videoB"

	cb1 := &Client{Hub: h, Name: "b1", Topic: topicB, Send: make(chan Message), Stats: NewClientStats()}
	cb2 := &Client{Hub: h, Name: "b2", Topic: topicB, Send: make(chan Message), Stats: NewClientStats()}
	cb3 := &Client{Hub: h, Name: "b3", Topic: topicB, Send: make(chan Message), Stats: NewClientStats()}

	topicC := "/videoC"

	cc1 := &Client{Hub: h, Name: "c1", Topic: topicC, Send: make(chan Message), Stats: NewClientStats()}
	cc2 := &Client{Hub: h, Name: "c2", Topic: topicC, Send: make(chan Message), Stats: NewClientStats()}
	cc3 := &Client{Hub: h, Name: "c3", Topic: topicC, Send: make(chan Message), Stats: NewClientStats()}

	topicD := "/videoD"

	cd1 := &Client{Hub: h, Name: "d1", Topic: topicD, Send: make(chan Message), Stats: NewClientStats()}
	cd2 := &Client{Hub: h, Name: "d2", Topic: topicD, Send: make(chan Message), Stats: NewClientStats()}
	cd3 := &Client{Hub: h, Name: "d3", Topic: topicD, Send: make(chan Message), Stats: NewClientStats()}

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

	msgSize := 1024

	contentA := make([]byte, msgSize*1)
	contentB := make([]byte, msgSize*4)
	contentC := make([]byte, msgSize*2)
	contentD := make([]byte, msgSize*1)

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

	// note that we'd update time.Now() on each send in production
	//mA := &Message{Data: contentA, Sender: *ca1, Sent: time.Now(), Type: 0}
	mB := &Message{Data: contentB, Sender: *cb1, Sent: time.Now(), Type: 0}
	mC := &Message{Data: contentC, Sender: *cc1, Sent: time.Now(), Type: 0}
	mD := &Message{Data: contentD, Sender: *cd1, Sent: time.Now(), Type: 0}

	rxCount := counter.New()

	iterations := 20

	duration := time.Duration(iterations+2)*(20)*time.Millisecond + time.Millisecond

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
		mB.Sent = time.Now()
		h.Broadcast <- *mB

		mD.Sent = time.Now()
		h.Broadcast <- *mD

		time.Sleep(5 * time.Millisecond)

		mC.Sent = time.Now()
		h.Broadcast <- *mC

		mD.Sent = time.Now()
		h.Broadcast <- *mD

		time.Sleep(5 * time.Millisecond)

		mD.Sent = time.Now()
		h.Broadcast <- *mD

		time.Sleep(5 * time.Millisecond)

		mC.Sent = time.Now()
		h.Broadcast <- *mC

		mD.Sent = time.Now()
		h.Broadcast <- *mD

		time.Sleep(5 * time.Millisecond)

	}

	time.Sleep(duration)

	desiredCount := iterations * 7 * 2 //two receivers per feed
	if rxCount.Read() != desiredCount {
		t.Error("Got wrong message count, wanted/got", desiredCount, rxCount.Read())
	}

	// Check Hub Stats

	expectedMeanBytes := (4.0*1 + 2.0*2 + 1.0*4) / 7 * float64(msgSize)

	if h.Stats.Bytes.Mean() > expectedMeanBytes+1 {
		t.Errorf("Message size stats are wrong, wanted %f got %f", expectedMeanBytes, h.Stats.Bytes.Mean())
	}
	if h.Stats.Bytes.Mean() < expectedMeanBytes-1 {
		t.Errorf("Message size stats are wrong, wanted %f got %f", expectedMeanBytes, h.Stats.Bytes.Mean())

	}

	expectedDt := 20e-3 / 7
	// arbitrary precision of 10%; assumed sufficient to catch major mistakes in calculating Dt
	if h.Stats.Dt.Mean() > expectedDt*1.1 {
		t.Errorf("Dt stats are wrong, wanted %f got %f", expectedDt, h.Stats.Dt.Mean())
	}
	if h.Stats.Dt.Mean() < expectedDt*0.9 {
		t.Errorf("Dt stats are wrong, wanted %f got %f", expectedDt, h.Stats.Dt.Mean())
	}

	// if the first message Dt is not discarded, then Dt stats are skewed due to large Dt going back to 1970
	// check for large maximum Dt
	if (h.Stats.Dt.Max() - 6e-3) > 0 {
		t.Errorf("Dt max is too large, wanted Dt < %f but got %f", 0.006, h.Stats.Dt.Max())
	}

	// latency is tied to system capability, but a fail here should alert to a potential performance problem

	// arbitrary choice of a ratio of 10, was seeing around 5
	latencyRatio := h.Stats.Latency.Max() / h.Stats.Latency.Mean()
	if latencyRatio > 10 {
		t.Errorf("Ratio of max:mean latency has exceeded 10: %f", latencyRatio)
	}

	if h.Stats.Latency.Mean() > 100e-6 {
		t.Errorf("Mean latency has exceed 100 microseconds: %f", h.Stats.Latency.Mean())
	}

	// Check inactive Tx stats

	if compareFloat64(ca1.Stats.Tx.Dt.Mean(), 1e-20) > 0 {
		t.Errorf("Inactive client Tx Dt stats are wrong, wanted 0ms but got %f\n", ca1.Stats.Tx.Dt.Mean())
	}

	if compareFloat64(ca1.Stats.Tx.Size.Mean(), 1e-20) > 0 {
		t.Errorf("Inactive client Tx Size stats are wrong, wanted 0 but got %f\n", ca1.Stats.Tx.Size.Mean())
	}

	last := time.Since(ca1.Stats.Tx.Last).Seconds()
	if last < 2 {
		t.Errorf("Inactive Client Tx last send time less than 2 sec, so wrong %fs\n", last)
	}

	// Check inactive RX stats

	if compareFloat64(ca2.Stats.Rx.Dt.Mean(), 1e-20) > 0 {
		t.Errorf("Inactive client Rx Dt stats are wrong, wanted 0ms but got %f\n", ca1.Stats.Tx.Dt.Mean())
	}

	if compareFloat64(ca2.Stats.Rx.Size.Mean(), 1e-20) > 0 {
		t.Errorf("Inactive client Rx Size stats are wrong, wanted 0 but got %f\n", ca1.Stats.Tx.Size.Mean())
	}

	last = time.Since(ca1.Stats.Rx.Last).Seconds()
	if last < 2 {
		t.Errorf("Inactive Client Rx last send time less than 2 sec, so wrong %fs\n", last)
	}

	// Check active TX stats

	if compareFloat64(math.Abs(cd1.Stats.Tx.Dt.Mean()-5e-3), 0.5e-3) > 0 {
		t.Errorf("Client Tx Dt stats are wrong, wanted 4.5ms < Dt < 5.5ms but got %f\n", cd1.Stats.Tx.Dt.Mean())
	}

	if compareFloat64(math.Abs(cd1.Stats.Tx.Size.Mean()-float64(msgSize)), 1) > 0 {
		t.Errorf("Client Tx Size stats are wrong, wanted %d < Size < %d but got %f\n", msgSize, msgSize, cd1.Stats.Tx.Size.Mean())
	}

	last = time.Since(cd1.Stats.Tx.Last).Seconds()
	if last > 1 {
		t.Errorf("Client Tx last send time exceeded 1s into the past %fs\n", last)
	}

	// Check active RX stats

	if compareFloat64(cd2.Stats.Rx.Dt.Mean(), 5.5e-3) > 0 {
		t.Errorf("Client Rx Dt stats are wrong, wanted 4.5ms < Dt < 5.5ms but got %f\n", cd2.Stats.Tx.Dt.Mean())
	}
	if compareFloat64(cd2.Stats.Rx.Dt.Mean(), 4.5e-3) < 0 {
		t.Errorf("Client Rx Dt stats are wrong, wanted 4.5ms < Dt < 5.5ms but got %f\n", cd2.Stats.Tx.Dt.Mean())
	}

	if int(math.Abs(cd2.Stats.Rx.Size.Mean()-float64(msgSize))) > 1 {
		t.Errorf("Client Rx Size stats are wrong, wanted %d < Size < %d but got %f\n", msgSize, msgSize, cd2.Stats.Tx.Size.Mean())
	}

	last = time.Since(cd2.Stats.Rx.Last).Seconds()
	if last > 1 {
		t.Errorf("Client Rx last send time exceeded 1s into the past %fs\n", last)
	}
	close(closed)

	fmt.Printf("-----------------------------------------\n")
	fmt.Printf("Average test bitrate: %0.1f Mbit/s\n", (h.Stats.Bytes.Mean()*float64(h.Stats.Bytes.Count()))/float64(duration.Seconds())/(1024*1024)*8)
	fmt.Printf("Average test latency: %0.1f microseconds\n", h.Stats.Latency.Mean()*1e6)
	fmt.Printf("-----------------------------------------\n")
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

func compareFloat64(a float64, b float64) int {

	aa := big.NewFloat(a)
	bb := big.NewFloat(b)

	return aa.Cmp(bb)

}
