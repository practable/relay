package vw

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func (app *App) startHTTP() {
	defer app.WaitGroup.Done()
	log.WithField("opts", app.Opts).Debug("http.Server looking at opts....")
	log.WithField("port", app.Opts.Port).Debug("http.Server listening port set")

	srv := app.startHTTPServer(app.Opts.Port)

	log.Debug("Started http.Server")

	<-app.Closed // wait for shutdown

	log.Debug("Starting to close http.Server")
	if err := srv.Shutdown(context.TODO()); err != nil {
		log.WithField("error", err).Fatal("Failure/timeout shutting down the http.Server gracefully")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(app.Opts.HTTPWaitMs)*time.Millisecond)
	defer cancel()

	srv.SetKeepAlivesEnabled(false)
	if err := srv.Shutdown(ctx); err != nil {
		log.WithField("error", err).Fatal("Could not gracefully shutdown http.Server")
	}

	log.Debug("Stopped http.Server")

} // startHTTP

func (app *App) startHTTPServer(port int) *http.Server {

	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{Addr: addr}

	var router = mux.NewRouter()

	// for profiler
	router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

	router.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	router.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	router.Handle("/debug/pprof/block", pprof.Handler("block"))

	router.HandleFunc("/api", app.handleAPI)
	router.HandleFunc("/api/destinations", app.handleDestinationAdd).Methods("PUT", "POST", "UPDATE")
	router.HandleFunc(`/api/destinations/{id:[a-zA-Z0-9\-\/]+}`, app.handleDestinationDelete).Methods("DELETE")
	router.HandleFunc("/api/destinations/all", app.handleDestinationShowAll).Methods("GET")
	router.HandleFunc("/api/destinations/all", app.handleDestinationDeleteAll).Methods("DELETE")
	router.HandleFunc(`/api/destinations/{id:[a-zA-Z0-9\-\/]+}`, app.handleDestinationShow).Methods("GET")
	router.HandleFunc("/api/streams", app.handleStreamAdd).Methods("PUT", "POST", "UPDATE")
	router.HandleFunc(`/api/streams/{stream:[a-zA-Z0-9\-\/]+}`, app.handleStreamDelete).Methods("DELETE")
	router.HandleFunc("/api/streams/all", app.handleStreamShowAll).Methods("GET")
	router.HandleFunc("/api/streams/all", app.handleStreamDeleteAll).Methods("DELETE")
	router.HandleFunc(`/api/streams/{stream:[a-zA-Z0-9\-\/]+}`, app.handleStreamShow).Methods("GET")
	router.HandleFunc("/healthcheck", app.handleHealthcheck).Methods("GET")
	router.HandleFunc(`/ts/{feed:[a-zA-Z0-9\-\/]+}`, app.handleTs)
	router.HandleFunc(`/ws/{feed:[a-zA-Z0-9\-\/]+}`, app.handleWs)

	srv.Handler = router

	go func() {
		//https://stackoverflow.com/questions/39320025/how-to-stop-http-listenandserve
		// returns ErrServerClosed on graceful close
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.WithField("error", err).Fatal("http.ListenAndServe")
		}
		log.Debug("Exiting http.Server")
	}()

	// returning reference so caller can call Shutdown()
	return srv
}
