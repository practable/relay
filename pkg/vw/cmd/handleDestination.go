package cmd

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/timdrysdale/relay/pkg/rwc"
)

// curl -X GET http://localhost:8888/api/destinations/all
func (app *App) handleDestinationShowAll(w http.ResponseWriter, r *http.Request) {

	output, err := json.Marshal(app.Websocket.Rules)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.Write(output)

}

// curl -X GET http://localhost:8888/api/destinations/01
func (app *App) handleDestinationShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	output, err := json.Marshal(app.Websocket.Rules[id])
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.Write(output)

}

/*  Add a new stream rule

Example:

curl -X POST -H "Content-Type: application/json" \
-d '{"stream":"/stream/front/large","feeds":["video0","audio0"]}'\
http://localhost:8888/api/streams/video

*/
func (app *App) handleDestinationAdd(w http.ResponseWriter, r *http.Request) {

	b, err := ioutil.ReadAll(r.Body)

	defer r.Body.Close()

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	var rule rwc.Rule
	err = json.Unmarshal(b, &rule)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	rule.Stream = strings.TrimPrefix(rule.Stream, "/") //to match trimming we do in handleStreamAdd

	app.Websocket.Add <- rule

	output, err := json.Marshal(rule)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.Write(output)
}

// curl -X DELETE http://localhost:8888/api/destinations/00
func (app *App) handleDestinationDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	app.Websocket.Delete <- id

	output, err := json.Marshal(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.Write(output)
}

func (app *App) handleDestinationDeleteAll(w http.ResponseWriter, r *http.Request) {

	id := "deleteAll"

	app.Websocket.Delete <- id

	output, err := json.Marshal(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.Write(output)
}
