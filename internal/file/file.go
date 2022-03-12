package file

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

// regexp for parsing a comment line
// if the first non-whitespace char on a line is # then it is ignored
// the first capture group is the rest of the line starting at the first
// non-whitespace character after the initial #
// lines starting with #, ##, ### etc are treated the same
// a + or - post fix indicates whether to echo the comment to the local output
// + for echo, - for do not echo. No + or - is considered a -, i.e. do not echo
const m = "^\\s*\\#+([+-]*)\\s*(.*)"

var mre = regexp.MustCompile(m)

// regexp for parsing a delay
/* note you can include a 's' after the delay value for readability, but no other duration indicator is accepted
e.g. these will pass the regexp (whether fractional minutes are valid is separate issue!)
[ 0.3s ] foo
[0.3s ] foo
[ 0.3s] foo
[0.3s] foo
[ 0.3 ] foo
[ 0.3 ] foo
[ 0.3] foo
[0.3 ] foo
[0.3] foo
[ 1h ] bar
[ 1h5.3m0.5s ] asdf
[] bar
*/

//^\s*\[\s*([a-zA-Z0-9.]*)\s*]\s*(.*)
const d = "^\\s*\\[\\s*([a-zA-Z0-9.]*)\\s*]\\s*(.*)"

var dre = regexp.MustCompile(d)

// regexp for parsing a condition in one pass (but misses malformed expressions)
// "^\s*\<\'([^']*)'\s*,\s*([0-9]*)\s*,\s*([0-9]*)\s*\>"
//const c = "^\\s*\\<\\'([^']*)'\\s*,\\s*([0-9]*)\\s*,\\s*([0-9]*)\\s*\\>"

// regexp for identifying a condition (needs a second step to parse the arguments)
const ci = "^\\s*<(.*)>\\s*(.*)"

var cire = regexp.MustCompile(ci)

// regexp for parsing the arguments to a condition
const ca = "^\\s*\\'([^']*)\\'\\s*,\\s*([0-9]*)\\s*,\\s*([0-9hmns\\.]*)\\s*"

var care = regexp.MustCompile(ca)

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
		s := scanner.Text()
		log.Debugf("Started parsing %s", s)
		out <- ParseLine(s)
	}

	close(out) //so receiver can range over channel

	return scanner.Err()

}

func ParseLine(line string) interface{} {

	defer func() {
		log.Debugf("Finished parsing %s", line)
	}()

	// comment
	if mre.MatchString(line) {

		log.Debugf("comment found in %s", line)

		m := mre.FindStringSubmatch(line)

		msg := m[2]

		echo := false

		if m[1] == "+" {
			echo = true
		}

		verb := "ignore"
		if echo {
			verb = "echo"
		}

		log.Infof("Parsed comment to %s: %s", verb, msg)

		return Comment{
			Msg:  msg,
			Echo: echo,
		}

	}

	if dre.MatchString(line) {

		log.Debugf("Delay command found in %s", line)

		d := dre.FindStringSubmatch(line)

		if len(d) < 2 {
			return Error{fmt.Sprintf("malformed delay command: %s", line)}
		}

		var t time.Duration

		if len(d[1]) > 1 {

			var err error //manage the scope of t

			t, err = time.ParseDuration(d[1])

			if err != nil {
				return Error{fmt.Sprintf("unknown delay time format: %s", line)}
			}
		}

		log.Infof("Parsed message to send after %s: %s", t, d[2])

		return Send{
			Msg:   d[2],
			Delay: t,
		}

	}

	if cire.MatchString(line) {

		log.Debugf("Condition command found in %s", line)

		c := cire.FindStringSubmatch(line)

		if len(c) < 3 {
			return Error{fmt.Sprintf("malformed condition command with only %d arguments (need 3): %s", len(c), line)}
		}

		args := care.FindStringSubmatch(c[1]) // the argument to the condition

		if len(args) < 4 {
			return Error{fmt.Sprintf("malformed condition command: %s", line)}
		}

		re, err := regexp.Compile(args[1])
		if err != nil {
			return Error{fmt.Sprintf("malformed condition command %s; first argument %s should be regexp pattern, but did not compile because %s. Line was %s", c, args[1], err.Error(), line)}
		}

		n, err := strconv.Atoi(args[2])
		if err != nil {
			return Error{fmt.Sprintf("malformed condition command %s; second argument %s should be integer, count of messages to await. Line was: %s", c, args[2], line)}
		}

		d, err := time.ParseDuration(args[3])
		if err != nil {
			return Error{fmt.Sprintf("malformed condition command %s; third argument %s should be timeout duration in format like 10s or 1m. Yours could not be parsed because %s. Line was was %s", c, args[3], err.Error(), line)}
		}

		log.Infof("Parsed message to send with condition to wait for %d results matching %s within %s: %s", n, args[1], d, c[2])
		return Send{
			Msg: c[2],
			Condition: Condition{
				Filter:  *re,
				Count:   n,
				Timeout: d,
			},
		}

	}

	return Send{
		Msg: line,
	}

}

type Comment struct {
	Msg  string
	Echo bool
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
	Filter  regexp.Regexp
	Count   int
	Timeout time.Duration
}
