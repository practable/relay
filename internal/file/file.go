package file

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"time"
)

// regexp for parsing a comment line
// if the first non-whitespace char on a line is # then it is ignored
const m = "^\\s*\\#"

var mr = regexp.MustCompile(m)

// regexp for parsing a delay
/* note you can include a 's' after the delay value for readability, but no other duration indicator is accepted
e.g. these are valid delays
[ 0.3 s] foo
[ 0.3s ] foo
[0.3s ] foo
[ 0.3s] foo
[0.3s] foo
[ 0.3 ] foo
[ 0.3 ] foo
[ 0.3] foo
[0.3 ] foo
[0.3] foo
[] bar

and these are valid comments to be echoed to output (as opposed to ignored)

[#] some comment
[ #] some comment
[# ] some comment
[ # ] some comment

*/

// const d = "^\\s*\\[\\s*([0-9]*\\.*[0-9]*)\\s*[s]*\\s*\\]\\s*(.*)" // no echo command
const d = "^\\s*\\[\\s*(\\#|[[0-9]*\\.*[0-9]*\\s*[s]*)\\s*\\]\\s*(.*)"

var dr = regexp.MustCompile(d)

// regexp for parsing a condition
const c = "\\s*\\<\\'([^']*)'\\s*,\\s*([0-9]*)\\s*,\\s*([0-9]*)\\s*\\>"

var cr = regexp.MustCompile(c)

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

type Send struct {
	Msg       string
	Delay     time.Duration
	Condition Condition
}

type Condition struct {
	Active  bool
	Filter  regexp.Regexp
	Count   int
	Timeout time.Duration
}
