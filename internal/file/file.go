package file

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/client9/reopen"
	"github.com/gorilla/websocket"
	"github.com/practable/relay/internal/reconws"
	log "github.com/sirupsen/logrus"
)

// Run connects to the session and handles writing to/from files
// If a playfilename is present, the connection is closed once the file has finished playing
// A long delay can be left at the end of the playfile to keep the connection open and logging
// Connections without a playfilename are kept open indefinitely.
// interval sets how often the condition check timeout is checked - this has a small effect on CPU
// usage, and can be set at say 10ms for testing, or 1s or more for usage with long collection periods
func Run(ctx context.Context, hup chan os.Signal, session, token, logfilename, playfilename string, interval time.Duration, check, force bool) error {

	var err error //manage scope of f

	var lines []interface{}

	// load our playfile, if we have one, and check for errors

	if len(playfilename) > 0 && playfilename != "-" {

		lines, err = LoadFile(playfilename)

		if err != nil {
			fmt.Printf("%s", err.Error())
			os.Exit(1)
		}

		errorsList, err := Check(lines)

		if err != nil {

			for _, str := range errorsList {
				log.Errorf("%s\n", str)
			}

			if !force {
				log.Errorf("%d errors detected checking play file", len(errorsList))
				return errors.New("Error(s) checking play file")
			}

			log.Infof("Errors detected in play file, but force=true so continuing")
		}
	}

	if check { // we've checked the file, let's return
		return nil
	}

	// playfile ok, or omitted, so set up log to file
	var f *reopen.FileWriter
	var logStdout bool

	if logfilename == "-" {

		logStdout = true

		// do nothing extra here. We we can ignore sighup for stdout
		// because re-opening stdout does nothing

	} else if len(logfilename) > 0 {

		f, err = reopen.NewFileWriter(logfilename)
		if err != nil {
			return err
		}

		done := make(chan struct{})
		defer close(done) //avoid leaking goro on exit

		// listen for sighup until we are done, or exiting
		go func() {
			log.Info("starting listening for sighup") //TODO demote to debug after fix
			for {
				select {
				case <-done:
					log.Info("done, finished playfile? No longer listening for signup")
					return // we've finished the playfile, most likely
				case <-ctx.Done():
					log.Info("Context cancelled, no longer listening for signup")
					return //avoid leaking this goroutine if we are cancelled
				case <-hup:
					log.Infof("SIGHUP detected, reopening LOG file %s\n", logfilename)
					f.Reopen()
				}
			}
		}()

	}

	// log into the session
	r := reconws.New()

	go r.ReconnectAuth(ctx, session, token)

	//channel for writing FilterActions
	a := make(chan FilterAction, 10)

	//channel for sending messages
	s := make(chan string, 10)

	// channel for writing to the file
	w := make(chan Line, 10) //add some buffering in case of a burst of messages

	// channel for incoming lines from websocket
	in := make(chan Line, 10)

	// convert format
	go WsMessageToLine(ctx, r.In, in)

	// split for use by conditionCheck and filter
	in0 := make(chan Line, 10)
	in1 := make(chan Line, 10)

	go Tee(ctx, in, in0, in1)

	// filter lines from in0 to w, if they pass the filter
	go FilterLines(ctx, a, in0, w)

	if logStdout {
		// write filtered lines from w to stdout
		go Write(ctx, w, os.Stdout)
	}
	// write filtered lines from w to f, if f has been specified and opened
	if f != nil {
		go Write(ctx, w, f)
	}

	c := make(chan ConditionCheck, 10)

	// monitor incoming messages and respond to
	// condition check requests from Play on c
	go ConditionCheckLines(ctx, c, in1, interval)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-s:
				r.Out <- reconws.WsMessage{Type: websocket.TextMessage, Data: []byte(msg)}
			}
		}
	}()

	if len(lines) > 0 {
		// play lines
		close := make(chan struct{})
		go Play(ctx, close, lines, a, s, c, w)
		<-close //Play closes close when it has finished playing the file

	} else {

		if playfilename == "-" {
			return errors.New("playing from stdin not implemented yet")
			//go PlayStdin(ctx, a, s)
		}
		// if no playing from file or stdin, simply wait so we can log incoming to file
		<-ctx.Done() //wait to be cancelled
	}

	return nil

}
