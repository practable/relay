package cmd

import (
	"fmt"
	"log"
	"net/http"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("indexHandler")
	log.Println(r.URL)
	if r.URL.Path != "/index.html" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "index.html")
}
