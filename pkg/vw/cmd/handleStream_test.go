package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// These tests do not start the hub or the websocket client
// Their channels can be read by the test code, saving mocking
// and simpler than inspecting the side effects of a running
// Hub and Websocket

func TestHandleStreamAdd(t *testing.T) {

	rule := []byte(`{"stream":"/stream/large","feeds":["audio","video0"]}`)

	req, err := http.NewRequest("PUT", "/api/streams", bytes.NewBuffer(rule))
	if err != nil {
		t.Error(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	a := testApp(false)
	handler := http.HandlerFunc(a.handleStreamAdd)

	go func() {
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		//note prefix / on stream is removed
		expected := `{"stream":"stream/large","feeds":["audio","video0"]}`
		if rr.Body.String() != expected {
			t.Errorf("handler returned unexpected body: got %v want %v",
				rr.Body.String(), expected)
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

func TestHandleStreamDelete(t *testing.T) {

	req, err := http.NewRequest("DELETE", "", nil)
	if err != nil {
		t.Error(err)
	}

	req = mux.SetURLVars(req, map[string]string{
		"stream": "video0",
	})

	rr := httptest.NewRecorder()

	a := testApp(false)
	handler := http.HandlerFunc(a.handleStreamDelete)

	go func() {
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}
	}()

	got := <-a.Hub.Delete

	if got != "video0" {
		t.Error("Wrong stream")
	}

}

func TestHandleStreamDeleteAll(t *testing.T) {

	req, err := http.NewRequest("DELETE", "", nil)
	if err != nil {
		t.Error(err)
	}

	req = mux.SetURLVars(req, map[string]string{
		"stream": "all",
	})

	rr := httptest.NewRecorder()

	a := testApp(false)
	handler := http.HandlerFunc(a.handleStreamDeleteAll)

	go func() {
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}
	}()

	got := <-a.Hub.Delete

	if got != "deleteAll" {
		t.Errorf("handler send wrong message on Hub.Delete: got %v want %v",
			got, "deleteAll")
	}

}

func TestHandleStreamShow(t *testing.T) {

	req, err := http.NewRequest("PUT", "", nil)
	if err != nil {
		t.Error(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{
		"stream": "stream/large",
	})

	rr := httptest.NewRecorder()

	a := testApp(false)
	handler := http.HandlerFunc(a.handleStreamShow)

	a.Hub.Rules = make(map[string][]string)
	a.Hub.Rules["stream/large"] = []string{"audio", "video0"}

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := "[\"audio\",\"video0\"]"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}

}

func TestHandleStreamShowAll(t *testing.T) {

	req, err := http.NewRequest("PUT", "", nil)
	if err != nil {
		t.Error(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	a := testApp(false)
	handler := http.HandlerFunc(a.handleStreamShowAll)

	a.Hub.Rules = make(map[string][]string)
	a.Hub.Rules["stream/large"] = []string{"audio", "video0"}
	a.Hub.Rules["stream/medium"] = []string{"audio", "video1"}

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := `{"stream/large":["audio","video0"],"stream/medium":["audio","video1"]}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}

}
