package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/timdrysdale/relay/pkg/rwc"
)

// These tests do not start the hub or the websocket client
// Their channels can be read by the test code, saving mocking
// and simpler than inspecting the side effects of a running
// Hub and Websocket

func TestHandleDestinationAdd(t *testing.T) {

	rule := []byte(`{"id":"00","stream":"/stream/large","destination":"wss://video.practable.io:443/large"}`)

	req, err := http.NewRequest("PUT", "", bytes.NewBuffer(rule))
	if err != nil {
		t.Error(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	a := testApp(false)
	handler := http.HandlerFunc(a.handleDestinationAdd)

	go func() {
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		// note prefix / on stream is removed
		expected := `{"id":"00","stream":"stream/large","destination":"wss://video.practable.io:443/large","token":""}`
		if rr.Body.String() != expected {
			t.Errorf("handler returned unexpected body: got %v want %v",
				rr.Body.String(), expected)
		}
	}()

	got := <-a.Websocket.Add

	if got.Stream != "stream/large" {
		t.Error("Wrong stream")
	}
	if got.Destination != "wss://video.practable.io:443/large" {
		t.Error("Wrong destination")
	}
	if got.Id != "00" {
		t.Error("Wrong Id")
	}

}

func TestHandleDestinationDelete(t *testing.T) {

	req, err := http.NewRequest("DELETE", "", nil)
	if err != nil {
		t.Error(err)
	}

	req = mux.SetURLVars(req, map[string]string{
		"id": "00",
	})

	rr := httptest.NewRecorder()

	a := testApp(false)
	handler := http.HandlerFunc(a.handleDestinationDelete)

	go func() {
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}
	}()

	got := <-a.Websocket.Delete

	if got != "00" {
		t.Error("Wrong Id")
	}

}

func TestHandleDestinationDeleteAll(t *testing.T) {

	req, err := http.NewRequest("DELETE", "", nil)
	if err != nil {
		t.Error(err)
	}

	req = mux.SetURLVars(req, map[string]string{
		"id": "all",
	})

	rr := httptest.NewRecorder()

	a := testApp(false)
	handler := http.HandlerFunc(a.handleDestinationDeleteAll)

	go func() {
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}
	}()

	got := <-a.Websocket.Delete

	if got != "deleteAll" {
		t.Errorf("handler send wrong message on Websocket.Delete: got %v want %v",
			got, "deleteAll")
	}

}

func TestHandleDestinationShow(t *testing.T) {

	req, err := http.NewRequest("PUT", "", nil)
	if err != nil {
		t.Error(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{
		"id": "00",
	})

	rr := httptest.NewRecorder()

	a := testApp(false)
	handler := http.HandlerFunc(a.handleDestinationShow)

	a.Websocket.Rules = make(map[string]rwc.Rule)
	a.Websocket.Rules["00"] = rwc.Rule{Destination: "wss://video.practable.io:443/large", Stream: "/stream/large", Id: "00"}

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := `{"id":"00","stream":"/stream/large","destination":"wss://video.practable.io:443/large","token":""}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}

}

func TestHandleDestinationShowAll(t *testing.T) {

	req, err := http.NewRequest("PUT", "", nil)
	if err != nil {
		t.Error(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	a := testApp(false)
	handler := http.HandlerFunc(a.handleDestinationShowAll)

	a.Websocket.Rules = make(map[string]rwc.Rule)
	a.Websocket.Rules["stream/large"] = rwc.Rule{Stream: "/stream/large",
		Destination: "wss://somewhere",
		Id:          "00"}
	a.Websocket.Rules["stream/medium"] = rwc.Rule{Stream: "/stream/medium",
		Destination: "wss://overthere",
		Id:          "01"}

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := `{"stream/large":{"id":"00","stream":"/stream/large","destination":"wss://somewhere","token":""},"stream/medium":{"id":"01","stream":"/stream/medium","destination":"wss://overthere","token":""}}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}

}
