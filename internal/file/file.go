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

// regexp for parsing a condition
const c = "\\s*\\<\\'([^']*)'\\s*,\\s*([0-9]*)\\s*,\\s*([0-9]*)\\s*\\>"

var cre = regexp.MustCompile(c)

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

	// comment
	if mre.MatchString(line) {
		m := mre.FindStringSubmatch(line)

		msg := m[2]

		echo := false

		if m[1] == "+" {
			echo = true
		}

		return Comment{
			Msg:  msg,
			Echo: echo,
		}

	}

	if dre.MatchString(line) {

		d := dre.FindStringSubmatch(line)

		msg := d[2]

		t, err := time.ParseDuration(d[1])

		if err != nil {
			return Error{fmt.Sprintf("unknown delay time format: %s", line)}
		}

		return Send{
			Msg:   msg,
			Delay: t,
		}

	}

	if cre.MatchString(line) {

	}

	return line

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
