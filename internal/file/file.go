package file

import (
	"context"
	"fmt"
	"os"

	"github.com/client9/reopen"
)

// Run connects to the session and handles writing to/from files
// If a playfilename is present, the connection is closed once the file has finished playing
// A long delay can be left at the end of the playfile to keep the connection open and logging
// Connections without a playfilename are kept open indefinitely.
func Run(ctx context.Context, hup chan os.Signal, session, token, logfilename, playfilename string) error {

	var f *reopen.FileWriter
	var bf reopen.Reopener

	var err error

	if logfilename == "-" {

		bf = reopen.Stdout

		// we can ignore sighup because re-opening stdout does nothing

	} else if len(logfilename) > 0 {

		f, err = reopen.NewFileWriter(logfilename)
		if err != nil {
			return err
		}

		bf = reopen.NewBufferedFileWriter(f)

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
					bf.Reopen()
				}
			}
		}()

	}

	// channel for writing to the file
	//w := make(chan Line, 10) //add some buffering in case of a burst of messages

	// log into the session

	// write filtered incoming to fp

	// open playfilename, if given

	// play contents to session & exit

	// or if no playfile, log indefinitely

	return nil

}
