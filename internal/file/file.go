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
				AcceptPattern: *re,
				Count:         n,
				Timeout:       d,
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
	AcceptPattern regexp.Regexp
	Count         int
	Timeout       time.Duration
}

type Filter struct {
	AcceptPatterns *map[*regexp.Regexp]struct{}
	DenyPatterns   *map[*regexp.Regexp]struct{}
}

// NewFilter returns a pointer to a new,
// initialised filter ready for use
func NewFilter() *Filter {

	f := &Filter{}
	f.init()
	return f
}

func (f *Filter) init() {
	ap := make(map[*regexp.Regexp]struct{})
	dp := make(map[*regexp.Regexp]struct{})
	f.AcceptPatterns = &ap
	f.DenyPatterns = &dp
}

// Reset replaces both AcceptPatterns and DenyPatterns
// with empty initialised maps, ready for use
func (f *Filter) Reset() {
	f.init()
}

// Pass returns whether or not a line should
// be passed by this filter
// which is true if the Filter
//

// Pass returns a bool indicating whether
// a line passes (true) or is blocked (false)
// by the filter
func (f *Filter) Pass(line string) bool {

	if f.AllPass() {
		return true
	}

	if f.Deny(line) {
		return false
	}

	if f.Accept(line) {
		return true
	}

	return false

}

// AllPass returns true if both AcceptPatterns and DenyPatterns
// are empty, i.e. all messages should pass.
// we do this for convenience and efficiency, rather than
// having an explict 'all pass' filter added to the AcceptList
// because we'd have to remove it the first time we add a filter
// and the second time we add a filter we'd have to check whether
// the first filter was the allpass one, and we might not know
// whether that was from initialisation or explicitly added by
// a user ....
func (f *Filter) AllPass() bool {
	return len(*f.AcceptPatterns) == 0 && len(*f.DenyPatterns) == 0
}

// match checks whether a string matches any patterns in the list of patterns
func match(line string, patterns *map[*regexp.Regexp]struct{}) bool {
	for p := range *patterns {
		if p.MatchString(line) {
			return true
		}
	}
	return false
}

// Deny returns true if this line is blocked by the filter
func (f *Filter) Deny(line string) bool {
	return match(line, f.DenyPatterns)
}

// Accept returns true if this line is passed by the filter
func (f *Filter) Accept(line string) bool {
	return match(line, f.AcceptPatterns)
}

// AddAcceptPattern adds a pattern to the AcceptPatterns
// that will be used to check if a message is accepted (passed)
func (f *Filter) AddAcceptPattern(p *regexp.Regexp) {
	(*f.AcceptPatterns)[p] = struct{}{}
}

// AddDenyPattern adds a pattern to the DenyPatterns
// that will be used to check if a message is denied (blocked)
func (f *Filter) AddDenyPattern(p *regexp.Regexp) {
	(*f.DenyPatterns)[p] = struct{}{}
}

// DeleteAcceptPattern will remove a given pattern from the
// list of patterns used to check for acceptance of a line
func (f *Filter) DeleteAcceptPattern(p *regexp.Regexp) {
	delete(*f.AcceptPatterns, p)
}

// DeleteDenyPattern will remove a given pattern from the
// list of patterns used to check for denial of a line
func (f *Filter) DeleteDenyPattern(p *regexp.Regexp) {
	delete(*f.DenyPatterns, p)
}
