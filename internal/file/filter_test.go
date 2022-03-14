package file

import (
	"bufio"
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFilterLines(t *testing.T) {

	setup := []FilterAction{
		FilterAction{
			Verb:    Accept,
			Pattern: regexp.MustCompile(`[a-h]`),
		},

		FilterAction{
			Verb:    Accept,
			Pattern: regexp.MustCompile(`[R-Z]`),
		},

		FilterAction{
			Verb:    Deny,
			Pattern: regexp.MustCompile(`[0-9]`),
		},

		FilterAction{
			Verb:    Deny,
			Pattern: regexp.MustCompile(`[#!&%]`),
		},
	}

	reset := FilterAction{
		Verb: Reset,
	}

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	// FilterLines will be running when we use these
	// so no need to buffer
	actions := make(chan FilterAction)
	in := make(chan Line)

	// buffer size >= number of actions/lines sent
	// because we read after sending
	out := make(chan Line, 10)

	go FilterLines(ctx, actions, in, out)

	for _, a := range setup {
		actions <- a
	}

	lines := []Line{
		Line{Content: "ah"},
		Line{Content: "ah#"},
		Line{Content: "ah0"},
		Line{Content: "Ah"}, //accepted because A is not denied, h is accepted
		Line{Content: "Az"},
		Line{Content: "abcd efg"},
	}

	// send messages
	for _, line := range lines {
		in <- line
	}

	expected := []Line{
		lines[0],
		lines[3],
		lines[5],
	}

	actual := []Line{}

READ0:
	for {
		select {
		case <-time.After(10 * time.Millisecond):
			break READ0
		case line := <-out:
			t.Logf("some-pass test: %s", line)
			actual = append(actual, line)
		}
	}

	assert.Equal(t, expected, actual)

	// reset, check all pass
	actions <- reset

	expected = lines

	// send messages
	for _, line := range lines {
		in <- line
	}

	actual = []Line{}

READ1:
	for {
		select {
		case <-time.After(10 * time.Millisecond):
			break READ1
		case line := <-out:
			t.Logf("all-pass test: %s", line)
			actual = append(actual, line)
		}
	}

	assert.Equal(t, expected, actual)

}

func TestFilter(t *testing.T) {

	p0 := regexp.MustCompile("[a-h]")
	p1 := regexp.MustCompile("[R-Z]")
	p2 := regexp.MustCompile("[0-9]")
	p3 := regexp.MustCompile("[#!&%]")

	f := NewFilter()

	f.AddAcceptPattern(p0)
	f.AddAcceptPattern(p1)
	f.AddDenyPattern(p2)
	f.AddDenyPattern(p3)

	s := `ach
ach0
ach!
TUV23
TUV%
TUV
ACH
tuv
`

	expected := []bool{
		true,
		false,
		false,
		false,
		false,
		true,
		false,
		false,
	}

	n := strings.Count(s, "\n")
	assert.Equal(t, len(expected), n)

	in := strings.NewReader(s)
	scanner := bufio.NewScanner(in)

	idx := 0
	for scanner.Scan() {
		line := scanner.Text()
		assert.Equal(t, expected[idx], f.Pass(line))
		idx++
	}

	// Check all message pass after reset
	f.Reset()
	in = strings.NewReader(s)
	scanner = bufio.NewScanner(in)

	idx = 0
	for scanner.Scan() {
		line := scanner.Text()
		assert.Equal(t, true, f.Pass(line))
		idx++
	}

	// Check whether delete works (have to add two filters
	// to avoid returning to allpass when you delete one)
	assert.Equal(t, 0, len(*f.AcceptPatterns))
	f.AddAcceptPattern(p0)
	f.AddAcceptPattern(p1)
	assert.Equal(t, 2, len(*f.AcceptPatterns))
	assert.True(t, f.Pass("abc"))

	f.DeleteAcceptPattern(p0)
	assert.Equal(t, 1, len(*f.AcceptPatterns))
	assert.False(t, f.Pass("abc"))

	// check deleteDenyPattern
	f.AddDenyPattern(p2)
	f.AddDenyPattern(p3)
	assert.False(t, f.Pass("T!"))
	f.DeleteDenyPattern(p3)
	assert.True(t, f.Pass("T!"))

	// check delete when a different regexp instance is used
	// to avoid implementations that don't compare the operation
	// of filters and instead use something inappropriate like
	// the value of a pointer etc. That would not work when
	// separate lines will each generate their own regexp
	// expressions rather than sharing a pointer
	f.AddDenyPattern(p3)
	assert.False(t, f.Pass("T!"))
	p3a := regexp.MustCompile("[#!&%]")
	f.DeleteDenyPattern(p3a)
	assert.True(t, f.Pass("T!"))

	f.AddAcceptPattern(p0)
	assert.True(t, f.Pass("abc"))
	p0a := regexp.MustCompile("[a-h]")
	f.DeleteAcceptPattern(p0a)
	assert.False(t, f.Pass("abc"))

	// check that only exact filters are deleted
	f.AddAcceptPattern(p0)
	assert.True(t, f.Pass("abc"))
	p5 := regexp.MustCompile("[a-h]\\s*")
	f.DeleteAcceptPattern(p5)
	assert.True(t, f.Pass("abc")) //passed because p0 not deleted

	//regexp.Regexp stores original text, so whitespace IS different - don't delete
	p0b := regexp.MustCompile(" [a-h]")
	f.DeleteAcceptPattern(p0b)
	assert.True(t, f.Pass("abc")) //passed because p0 not deleted

}
