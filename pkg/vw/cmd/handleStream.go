package cmd

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/relay/pkg/agg"
)

func (app *App) handleStreamShowAll(w http.ResponseWriter, r *http.Request) {
	output, err := json.Marshal(app.Hub.Rules)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("content-type", "application/json")
	_, err = w.Write(output)
	if err != nil {
		log.Errorln(err)
	}
}

func (app *App) handleStreamShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stream := vars["stream"]

	if feeds, ok := app.Hub.Rules[stream]; ok {

		output, err := json.Marshal(feeds)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("content-type", "application/json")
		w.Write(output)
	} else {
		http.Error(w, "Stream not found", 404)
		return
	}

}

/*  Add a new stream rule

Example:

curl -X POST -H "Content-Type: application/json" \
-d '{"stream":"/stream/front/large","feeds":["video0","audio0"]}'\
http://localhost:8888/api/streams/video

*/
func (app *App) handleStreamAdd(w http.ResponseWriter, r *http.Request) {

	b, err := ioutil.ReadAll(r.Body)

	defer r.Body.Close()

	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}
	var rule agg.Rule
	err = json.Unmarshal(b, &rule)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	rule.Stream = strings.TrimPrefix(rule.Stream, "/") //can't delete a stream registered with leading prefix

	app.Hub.Add <- rule

	output, err := json.Marshal(rule)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.Write(output)
}

func (app *App) handleStreamDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stream := vars["stream"]

	app.Hub.Delete <- stream

	output, err := json.Marshal(stream)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.Write(output)
}

func (app *App) handleStreamDeleteAll(w http.ResponseWriter, r *http.Request) {

	stream := "deleteAll"

	app.Hub.Delete <- stream

	output, err := json.Marshal(stream)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.Write(output)

}
