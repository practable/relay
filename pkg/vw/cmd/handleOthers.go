package cmd

import (
	"encoding/json"
	"net/http"
)

func (app *App) handleHealthcheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("ok")
}
