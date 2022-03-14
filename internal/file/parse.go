package file

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// ConditionCheckLines monitors incoming Lines and uses them to assess whether a current condition has been
// met, which is expressed as Count lines maching the AcceptPattern occuring before the timeout.
// The Satisfied channel is closed regardless of completion or timeout - there is no indication from
// ConditionCheckLines whether the condition was satisfied. This is left to any analysis task that the
// user is presumably coding up separately to analyse the results. In any case, for checking equipment,
// the condition is normally used to ensure a sufficiency of data with some tolerance, so not exactly
// meeting the condition is not as much as an issue as it might otherwise be - hence we can leave this
// to the external analysis to decide.
func ConditionCheckLines(ctx context.Context, cc chan ConditionCheck, in chan Line, interval time.Duration) {

	var checking bool //true if we get a new command
	var lines []Line  //store what we record
	var current ConditionCheck
	var stop time.Time

	go func() {
		for {
			select {

			case <-ctx.Done():
				return

			case <-time.After(interval):
				if checking {
					// check if we have timed out
					if time.Now().After(stop) {
						log.Infof("condition %s satisfied by timeout check at %s interval", current.Condition.String(), interval)
						checking = false
						close(current.Satisfied)
						current = ConditionCheck{} //prevent double close
						lines = []Line{}           //delete lines recorded
					}
				}
			case line := <-in:
				if checking {

					if current.Condition.AcceptPattern.MatchString(line.Content) {
						log.Debugf("accepted lines: %d, want %d", len(lines), current.Condition.Count)
						lines = append(lines, line)
					} else {
						log.Debugf("ignoring line, does not match")
					}

					if len(lines) >= current.Condition.Count {
						// we've got enough lines
						log.Infof("condition %s satisfied by receiving enough accepted lines", current.Condition.String())
						for i, line := range lines {
							log.Debugf("%d: %s", i, line.Content)
						}
						checking = false
						close(current.Satisfied)
						current = ConditionCheck{} //prevent double close
						lines = []Line{}           //delete lines recorded
					}

				}
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case c := <-cc:
			current = c
			checking = true
			stop = time.Now().Add(c.Condition.Timeout)

		}
	}
}

// Play takes a slice of Line and plays each Line as required, e.g. Wait, Send (with Delay or Condition), Comment (e.g. echo to file)
func Play(ctx context.Context, closed chan struct{}, lines []interface{}, a chan FilterAction, s chan string, c chan ConditionCheck, w chan Line) {

	defer close(closed) //signal we're done

	for idx, line := range lines {

		log.Debugf("%d (%T)\n", idx, line)

		switch line.(type) {
		case Comment:
			if c, ok := line.(Comment); ok {
				if c.Echo {
					w <- Line{
						Content: c.Msg,
						Time:    time.Now(),
					}
				}
			}
		case Error:
			// ignore it
		case Wait:
			if wait, ok := line.(Wait); ok {
				<-time.After(wait.Delay)
			}
		case Send:
			if send, ok := line.(Send); ok {
				// wait
				<-time.After(send.Delay)
				// see if there is a condition
				if CompleteCondition(send.Condition) {
					satisfied := make(chan struct{})
					c <- ConditionCheck{
						Satisfied: satisfied,
						Condition: send.Condition,
					}
					<-satisfied //wait until, maybe forever (some users may set very long values here, days, weeks etc)
				}
				s <- send.Msg
			}
		case FilterAction:
			if fa, ok := line.(FilterAction); ok {
				a <- fa
			}

		}

	}

}

// CompleteCondition returns true if all parts of the CompleteCondition
// are holding non-nil equivalent values.
func CompleteCondition(c Condition) bool {

	p := c.AcceptPattern.String() == ""
	n := c.Count <= 0
	t := c.Timeout == 0

	return !p && !n && !t
}

// Check returns an error if any Error structs are present in the slice of lines
func Check(lines []interface{}) ([]string, error) {

	var err error

	errors := []string{}

	for _, line := range lines {

		switch line.(type) {

		case Error:
			if e, ok := line.(Error); ok {
				errors = append(errors, e.string)
				err = fmt.Errorf("Found %d errors", len(errors))
			}
		}
	}

	return errors, err
}

// LoadFile reads a .play file and returns a slice of interfaces
// each one representing a line in the file
func LoadFile(filename string) ([]interface{}, error) {

	var lines []interface{}

	out := make(chan interface{}, 10)

	go func() {
		for {
			line, ok := <-out

			if !ok {
				return //avoid leaking this goroutine
			}

			lines = append(lines, line)
		}
	}()

	err := ParseFile(filename, out)

	return lines, err

}

// ParseFile parses a file into an interface per line,
// which is sent over the out channel.
func ParseFile(filename string, out chan interface{}) error {

	f, err := os.Open(filename)
	if err != nil {
		return err
	}

	defer f.Close()

	return ParseByLine(f, out)
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

// ParseLine parses a line and returns a struct representing it,
// e.g. Wait, Error, Comment, Send.
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

		if len(d[2]) > 0 {

			log.Infof("Parsed message to send after %s: %s", t, d[2])

			return Send{
				Msg:   d[2],
				Delay: t,
			}
		}

		log.Infof("Parsed wait for %s", t)
		return Wait{
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

	if fre.MatchString(line) {

		log.Debugf("Filter command found in %s", line)

		args := fre.FindStringSubmatch(line)

		if len(args) < 3 {
			return Error{fmt.Sprintf("malformed filter command: %s", line)}
		}

		var verb FilterVerb

		switch strings.ToLower(args[1]) {
		case "-", "d", "deny":
			verb = Deny
		case "+", "a", "accept":
			verb = Accept
		case "r", "reset":
			verb = Reset
		}

		switch verb {
		case Unknown:
			return Error{fmt.Sprintf("malformed filter command; first argument not one of [+,-,a,d,r,accept,deny,reset], but was %s", args[1])}
		case Reset:
			return FilterAction{
				Verb: Reset,
			}
		case Accept, Deny:
			re, err := regexp.Compile(args[2])
			if err != nil {
				return Error{fmt.Sprintf("malformed filter command; last argument %s should be regexp pattern, but did not compile because %s. Line was %s", args[2], err.Error(), line)}
			}
			return FilterAction{
				Verb:    verb,
				Pattern: re,
			}
		}

	}

	return Send{
		Msg: line,
	}

}
