package vw

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func (app *App) handleHealthcheck(w http.ResponseWriter, r *http.Request) {
	err := json.NewEncoder(w).Encode("ok")
	if err != nil {
		log.Errorf("error encoding healthcheck ok %s", err.Error())
	}
}
