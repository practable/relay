package file

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/client9/reopen"
	"github.com/practable/relay/internal/reconws"
	log "github.com/sirupsen/logrus"
)

// Run connects to the session and handles writing to/from files
// If a playfilename is present, the connection is closed once the file has finished playing
// A long delay can be left at the end of the playfile to keep the connection open and logging
// Connections without a playfilename are kept open indefinitely.
// interval sets how often the condition check timeout is checked - this has a small effect on CPU
// usage, and can be set at say 10ms for testing, or 1s or more for usage with long collection periods
func Run(ctx context.Context, hup chan os.Signal, connected chan struct{}, session, token, filename string, binary bool) error {

	var err error //manage scope of f

	// set up log to file
	var f *reopen.FileWriter
	// use empty filename or filename of "-" to mean stdout
	if len(filename) > 0 && filename != "-" {

		// open file for writing{

		f, err = reopen.NewFileWriter(filename)
		if err != nil {
			return err
		}

		// listen for sighup until we are done, or exiting
		go func() {
			log.Debug("starting listening for sighup")
			for {
				select {
				case <-ctx.Done():
					log.Debug("Context cancelled, no longer listening for signup")
					return //avoid leaking this goroutine if we are cancelled
				case <-hup:
					log.Infof("SIGHUP detected, reopening file %s", filename)
					err := f.Reopen()
					if err != nil {
						log.Errorf("error reopening file: %s", err.Error())
					}
				}
			}
		}()

	}

	// log into the session
	r := reconws.New()

	start := time.Now()
	go r.ReconnectAuth(ctx, session, token)

	select {
	case <-r.Connected: //wait for connection to be made
		close(connected) // used for testing (don't need handle reconnections in testing, so no need to renew the channel)
	case <-ctx.Done():
		return errors.New("context cancelled before connection established")
	}

	log.Infof("file.Run(): connected to session %s in %s", session, time.Since(start).String())

	// channel for incoming lines from websocket
	in := make(chan Line, 10)

	// convert format
	if binary {
		// do NOT format lines (else it corrupts binary data such as video feed)
		if f != nil {
			go WriteBinary(ctx, r.In, f)
		} else {
			go WriteBinary(ctx, r.In, os.Stdout)
		}
	} else {
		// format lines
		go WsMessageToLine(ctx, r.In, in)
		// write formatted lines to file or stdout
		if f != nil {
			go Write(ctx, in, f)
		} else {
			go Write(ctx, in, os.Stdout)
		}
	}

	<-ctx.Done() //wait to be cancelled

	return nil

}
