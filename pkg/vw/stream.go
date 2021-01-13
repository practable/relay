package vw

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"runtime/pprof"
	"time"

	"github.com/timdrysdale/relay/pkg/agg"
	"github.com/timdrysdale/relay/pkg/rwc"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

var app App

func Stream() {

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
		}
	}()

	//Websocket has to be instantiated AFTER the Hub
	app = App{Hub: agg.New(), Closed: make(chan struct{})}
	app.Websocket = rwc.New(app.Hub)

	// load configuration from environment variables VW_<var>
	if err := envconfig.Process("vw", &app.Opts); err != nil {
		log.Fatal("Configuration Failed", err.Error())
	}

	if app.Opts.CpuProfile != "" {

		f, err := os.Create(app.Opts.CpuProfile)

		if err != nil {
			log.WithField("error", err).Fatal("Could not create CPU profile")
		}

		defer f.Close()

		if err := pprof.StartCPUProfile(f); err != nil {
			log.WithField("error", err).Fatal("Could not start CPU profile")
		}

		go func() {

			time.Sleep(30 * time.Second)
			pprof.StopCPUProfile()

		}()
	}

	//set log format
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(sanitiseLevel(app.Opts.LogLevel))

	//log configuration
	log.WithField("s", app.Opts).Info("Specification")

	// trap SIGINT
	channelSignal := make(chan os.Signal, 1)
	signal.Notify(channelSignal, os.Interrupt)
	go func() {
		for range channelSignal {
			close(app.Closed)
			app.WaitGroup.Wait()
			os.Exit(1)
		}
	}()

	//TODO add waitgroup into agg/hub and rwc

	go app.Hub.RunWithStats(app.Closed)

	go app.Websocket.Run(app.Closed)

	go app.internalAPI("api")

	if app.Opts.API != "" {
		app.Websocket.Add <- rwc.Rule{Stream: "api", Destination: app.Opts.API, Id: "apiRule"}
	}

	app.WaitGroup.Add(1)
	go app.startHttp()

	// take it easy, pal
	app.WaitGroup.Wait()

}
