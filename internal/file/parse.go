package file

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

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
