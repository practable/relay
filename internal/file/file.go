package file

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/client9/reopen"
	"github.com/practable/relay/internal/reconws"
)

// Run connects to the session and handles writing to/from files
// If a playfilename is present, the connection is closed once the file has finished playing
// A long delay can be left at the end of the playfile to keep the connection open and logging
// Connections without a playfilename are kept open indefinitely.
func Run(ctx context.Context, hup chan os.Signal, session, token, logfilename, playfilename string) error {

	var err error //manage scope of f

	var lines []interface{}

	// load our playfile, if we have one, and check for errors

	if len(playfilename) > 0 && playfilename != "-" {

		lines, err = LoadFile(playfilename)

		if err != nil {
			fmt.Printf("%s", err.Error())
			os.Exit(1)
		}

		errors, err := Check(lines)

		if err != nil {

			for _, str := range errors {
				fmt.Printf("%s\n", str)
			}

			os.Exit(1)
		}
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
			for {
				select {
				case <-done:
					return // we've finished the playfile, most likely
				case <-ctx.Done():
					return //avoid leaking this goroutine if we are cancelled
				case <-hup:
					fmt.Printf("SIGHUP detected, reopening LOG file %s\n", logfilename)
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

	//channel for sending messages (wrapped in different types to suit line content)
	s := make(chan interface{}, 10)

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
	} else {
		// write filtered lines from w to f
		go Write(ctx, w, f)
	}

	c := make(chan ConditionCheck, 10)

	// monitor incoming messages and respond to
	// condition check requests from Play on c
	go ConditionCheckLines(ctx, c, in1)

	if len(lines) > 0 {
		// play lines
		close := make(chan struct{})
		go Play(ctx, close, lines, a, s, c)
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
