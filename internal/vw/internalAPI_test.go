package vw

import (
	"reflect"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/practable/relay/internal/agg"
	"github.com/practable/relay/internal/hub"
	"github.com/practable/relay/internal/rwc"
)

// Commands that we are testing ...
// {"verb":"add","what":"destination","rule":{"stream":"video0","destination":"wss://<some.relay.server>/in/video0","id":"0"}}
// {"verb":"delete","what":"destination","which":"<id>">}
// {"verb":"list","what":"destination","which":"all"}
// {"verb":"list","what":"destination","which":"<id>">}

// {"verb":"add","what":"stream","rule":{"stream":"video0","feeds":["video0","audio0"]}}
// {"verb":"delete","what":"stream","which":"<which>"}
// {"verb":"list","what":"stream","which":"all"}
// {"verb":"list","what":"stream","which":"<name>"}

// do one test with the internalAPI to check it is wired up ok, then
// test the handler directly for the rest of the tests
func TestInternalAPICommunicates(t *testing.T) {

	app = App{Hub: agg.New(), Closed: make(chan struct{})}
	app.Websocket = rwc.New(app.Hub)

	name := "api"
	go app.internalAPI(name)

	client, ok := <-app.Hub.Register

	if !ok {
		t.Errorf("Problem receiving internalAPI registration")
	}

	if client.Topic != name {
		t.Errorf("internalAPI registered with wrong name (%s/%s)\n", name, client.Topic)
	}

	cmd := []byte(`{"verb":"list","what":"destination","which":"all"}`)

	go func() {
		client.Send <- hub.Message{Sender: hub.Client{}, Data: cmd, Type: websocket.TextMessage, Sent: time.Now()}
	}()

	time.Sleep(1 * time.Millisecond)

	select {
	case msg, ok := <-client.Hub.Broadcast:
		if ok {
			if string(msg.Data) != "{}" {
				t.Error("Unexpected reply from internalAPI")
			}
		} else {
			t.Error("Problem with messaging channel")
		}

	case <-time.After(1 * time.Millisecond):
		t.Error("timeout waiting for internalAPI to reply")
	}

	close(app.Closed)
}

func TestInternalAPIBadCommand(t *testing.T) {

	a := testApp(false)

	cmd := []byte(`Not even JSON`)

	expected := errBadCommand

	_, err := a.handleAdminMessage(cmd)
	if err == nil {
		t.Error("Failed to throw error")
	} else if !reflect.DeepEqual(expected, err) {
		t.Errorf("Got wrong err %s/%s\n", expected, err)
	}

}

func TestInternalAPIHealthCheck(t *testing.T) {

	a := testApp(false)

	cmd := []byte(`{"verb":"healthcheck"}`)

	expected := []byte(`{"healthcheck":"ok"}`)

	reply, err := a.handleAdminMessage(cmd)
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	} else if !reflect.DeepEqual(expected, reply) {
		t.Errorf("Got wrong err %s/%s\n", expected, reply)
	}

}

func TestInternalAPIDestinationAdd(t *testing.T) {

	a := testApp(false)

	rule := `{"id":"00","stream":"/stream/large","destination":"wss://video.practable.io:443/large"}`
	cmd := []byte(`{"verb":"add","what":"destination","rule":` + rule + `}`)

	// note prefix / on stream is removed
	expected := []byte(`{"id":"00","stream":"stream/large","destination":"wss://video.practable.io:443/large","token":"","file":""}`)

	go func() {
		reply, err := a.handleAdminMessage(cmd)
		if err != nil {
			t.Error("unexpected error")
			return
		}
		if !reflect.DeepEqual(expected, reply) {
			t.Errorf("Got wrong rule %s/%s\n", expected, reply)
		}
	}()

	got := <-a.Websocket.Add

	if got.Stream != "stream/large" {
		t.Error("Wrong stream")
	}
	if got.Destination != "wss://video.practable.io:443/large" {
		t.Error("Wrong destination")
	}
	if got.ID != "00" {
		t.Error("Wrong ID")
	}
}

func TestInternalAPIDestinationDelete(t *testing.T) {

	a := testApp(false)

	cmd := []byte(`{"verb":"delete","what":"destination","which":"00"}`)

	// note prefix / on stream is removed
	expected := []byte(`{"deleted":"00"}`)

	go func() {
		reply, err := a.handleAdminMessage(cmd)
		if err != nil {
			t.Error("unexpected error")
			return
		}
		if !reflect.DeepEqual(expected, reply) {
			t.Errorf("Got wrong rule %s/%s\n", expected, reply)
		}
	}()

	got := <-a.Websocket.Delete

	if got != "00" {
		t.Error("Wrong ID")
	}
}

func TestInternalAPIDestinationDeleteAPIRule(t *testing.T) {

	a := testApp(false)

	cmd := []byte(`{"verb":"delete","what":"destination","which":"apiRule"}`)

	// note prefix / on stream is removed
	expected := errNoDeleteAPIRule //will be put into error message by internalAPI

	_, err := a.handleAdminMessage(cmd)
	if err == nil {
		t.Error("Failed to throw error")
		return
	}
	if !reflect.DeepEqual(expected, err) {
		t.Errorf("Got wrong response %s/%s\n", expected, err)
	}
}

func TestInternalAPIDestinationDeleteAll(t *testing.T) {

	a := testApp(false)
	a.Opts.API = "wss://some.relay.server:443/bi/some/where/unique"
	cmd := []byte(`{"verb":"delete","what":"destination","which":"all"}`)

	// note prefix / on stream is removed
	expected := []byte(`{"deleted":"deleteAll"}`)

	go func() {
		reply, err := a.handleAdminMessage(cmd)
		if err != nil {
			t.Error("unexpected error")
			return
		}
		if !reflect.DeepEqual(expected, reply) {
			t.Errorf("Got wrong rule %s/%s\n", expected, reply)
		}
	}()

	got := <-a.Websocket.Delete

	if got != "deleteAll" {
		t.Error("Wrong ID")
	}

	added := <-a.Websocket.Add

	if added.ID != "apiRule" {
		t.Error("Did not reinstate apiRule")
	}
	if added.Destination != a.Opts.API {
		t.Error("Did not reinstate apiRule with correct address")
	}
}

func TestInternalAPIDestinationShow(t *testing.T) {

	a := testApp(false)
	a.Websocket.Rules = make(map[string]rwc.Rule)
	a.Websocket.Rules["00"] = rwc.Rule{Destination: "wss://video.practable.io:443/large", Stream: "stream/large", ID: "00"}

	cmd := []byte(`{"verb":"list","what":"destination","which":"00"}`)
	expected := []byte(`{"id":"00","stream":"stream/large","destination":"wss://video.practable.io:443/large","token":"","file":""}`)

	reply, err := a.handleAdminMessage(cmd)
	if err != nil {
		t.Error("unexpected error")
		return
	}
	if !reflect.DeepEqual(expected, reply) {
		t.Errorf("Got wrong rule %s/%s\n", expected, reply)
	}

}

func TestInternalAPIDestinationShowAll(t *testing.T) {

	a := testApp(false)
	a.Websocket.Rules = make(map[string]rwc.Rule)
	a.Websocket.Rules["stream/large"] = rwc.Rule{Stream: "stream/large",
		Destination: "wss://somewhere",
		ID:          "00"}
	a.Websocket.Rules["stream/medium"] = rwc.Rule{Stream: "stream/medium",
		Destination: "wss://overthere",
		ID:          "01"}

	cmd := []byte(`{"verb":"list","what":"destination","which":"all"}`)
	expected := []byte(`{"stream/large":{"id":"00","stream":"stream/large","destination":"wss://somewhere","token":"","file":""},"stream/medium":{"id":"01","stream":"stream/medium","destination":"wss://overthere","token":"","file":""}}`)

	reply, err := a.handleAdminMessage(cmd)
	if err != nil {
		t.Error("unexpected error")
		return
	}
	if !reflect.DeepEqual(expected, reply) {
		t.Errorf("Got wrong rule %s/%s\n", expected, reply)
	}

}

// Streams

func TestInternalAPIStreamAdd(t *testing.T) {

	a := testApp(false)

	rule := `{"stream":"/stream/large","feeds":["audio","video0"]}`
	cmd := []byte(`{"verb":"add","what":"stream","rule":` + rule + `}`)

	// note prefix / on stream is removed
	expected := []byte(`{"stream":"stream/large","feeds":["audio","video0"]}`)

	go func() {
		reply, err := a.handleAdminMessage(cmd)
		if err != nil {
			t.Error("unexpected error")
			return
		}
		if !reflect.DeepEqual(expected, reply) {
			t.Errorf("Got wrong rule %s/%s\n", expected, reply)
		}
	}()
	got := <-a.Hub.Add

	if got.Stream != "stream/large" {
		t.Error("Wrong stream")
	}

	if got.Feeds[0] != "audio" {
		t.Error("Wrong feeds")
	}
	if got.Feeds[1] != "video0" {
		t.Error("Wrong feeds")
	}
}

func TestInternalAPIStreamDelete(t *testing.T) {

	a := testApp(false)

	cmd := []byte(`{"verb":"delete","what":"stream","which":"video0"}`)

	// note prefix / on stream is removed
	expected := []byte(`{"deleted":"video0"}`)

	go func() {
		reply, err := a.handleAdminMessage(cmd)
		if err != nil {
			t.Error("unexpected error")
			return
		}
		if !reflect.DeepEqual(expected, reply) {
			t.Errorf("Got wrong rule %s/%s\n", expected, reply)
		}
	}()

	got := <-a.Hub.Delete

	if got != "video0" {
		t.Error("Wrong stream")
	}

}

func TestInternalAPIStreamDeleteAll(t *testing.T) {

	a := testApp(false)

	cmd := []byte(`{"verb":"delete","what":"stream","which":"all"}`)

	// note prefix / on stream is removed
	expected := []byte(`{"deleted":"deleteAll"}`)

	go func() {
		reply, err := a.handleAdminMessage(cmd)
		if err != nil {
			t.Error("unexpected error")
			return
		}
		if !reflect.DeepEqual(expected, reply) {
			t.Errorf("Got wrong rule %s/%s\n", expected, reply)
		}
	}()

	got := <-a.Hub.Delete

	if got != "deleteAll" {
		t.Errorf("handler send wrong message on Hub.Delete: got %v want %v",
			got, "deleteAll")
	}
}

func TestInternalAPIStreamShow(t *testing.T) {

	a := testApp(false)
	a.Hub.Rules = make(map[string][]string)
	a.Hub.Rules["stream/large"] = []string{"audio", "video0"}

	cmd := []byte(`{"verb":"list","what":"stream","which":"stream/large"}`)
	expected := []byte(`{"feeds":["audio","video0"]}`)

	reply, err := a.handleAdminMessage(cmd)
	if err != nil {
		t.Error("unexpected error")
		return
	}
	if !reflect.DeepEqual(expected, reply) {
		t.Errorf("Got wrong rule %s/%s\n", expected, reply)
	}
}

func TestInternalAPIStreamShowAll(t *testing.T) {

	a := testApp(false)
	a.Hub.Rules = make(map[string][]string)
	a.Hub.Rules["stream/large"] = []string{"audio", "video0"}
	a.Hub.Rules["stream/medium"] = []string{"audio", "video1"}

	cmd := []byte(`{"verb":"list","what":"stream","which":"all"}`)
	expected := []byte(`{"stream/large":["audio","video0"],"stream/medium":["audio","video1"]}`)

	reply, err := a.handleAdminMessage(cmd)
	if err != nil {
		t.Error("unexpected error")
		return
	}
	if !reflect.DeepEqual(expected, reply) {
		t.Errorf("Got wrong rule %s/%s\n", expected, reply)
	}

}
