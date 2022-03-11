package file

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
)

// Run connects to the session and handles writing to/from files
func Run(ctx context.Context, hup chan os.Signal, session, token, logfilename, playfilename string) {

	go func() {
		for {
			select {
			case <-ctx.Done():
				return //avoid leaking this goroutine if we are cancelled
			case <-hup:
				fmt.Printf("SIGHUP detected, reopening LOG file %s\n", logfilename)
			}
		}
	}()

}

// ParseByLine reads from the supplied io.Reader, line by line,
// parsing each line into a struct representing known actions
// or errors, all of which are returned over out channel
func ParseByLine(in io.Reader, out chan interface{}) error {

	scanner := bufio.NewScanner(in)

	for scanner.Scan() {
		out <- ParseLine(scanner.Text())
	}

	close(out) //so receiver can range over channel

	return scanner.Err()

}

func ParseLine(line string) interface{} {

	var r interface{}

	r = Error{fmt.Sprintf("unknown line format: %s", line)}

	return r

}

type Error struct {
	string
}
