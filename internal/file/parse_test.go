package file

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConditionCheckLines(t *testing.T) {

	//we need to timeout, and line count tests
	//<'^foo\s*',5,100ms> {"send":"foos"}
	// test 0: send nothing, must timeout (and not hang)
	// test 1: send 6 "bar", must timeout (must check message against AcceptPattern)
	// test 2: send 4 "foo", must timeout (must count accurately)
	// test 3: send 5 "foo", must return before timeout (must count accurately)
	// test 4: send 5 "foo", THEN send condition, then wait: must timeout (don't use previous)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // we can run all tests without restarting ConditionCheckLines so just defer

	cc := make(chan ConditionCheck) // don't buffer, to keep some control of timing of check commands

	in := make(chan Line, 10) //buffer to simplify test writing

	go ConditionCheckLines(ctx, cc, in, 10*time.Millisecond)

	timeout := 150 * time.Millisecond

	cond := Condition{
		AcceptPattern: *regexp.MustCompile(`^foo\s*`),
		Count:         5,
		Timeout:       timeout,
	}

	ccs := []ConditionCheck{
		ConditionCheck{
			Satisfied: make(chan struct{}),
			Condition: cond,
		},
		ConditionCheck{
			Satisfied: make(chan struct{}),
			Condition: cond,
		},
		ConditionCheck{
			Satisfied: make(chan struct{}),
			Condition: cond,
		},
		ConditionCheck{
			Satisfied: make(chan struct{}),
			Condition: cond,
		},
		ConditionCheck{
			Satisfied: make(chan struct{}),
			Condition: cond,
		},
	}

	shouldTimeout := []bool{
		true,
		true,
		true,
		false,
		true,
	}

	preLines := [][]Line{
		[]Line{},
		[]Line{},
		[]Line{},
		[]Line{},
		[]Line{
			Line{Content: "foo"},
			Line{Content: "foo"},
			Line{Content: "foo"},
			Line{Content: "foo"},
			Line{Content: "foo"},
		},
	}

	postLines := [][]Line{
		[]Line{},
		[]Line{
			Line{Content: "bar"},
			Line{Content: "bar"},
			Line{Content: "bar"},
			Line{Content: "bar"},
			Line{Content: "bar"},
		},
		[]Line{
			Line{Content: "foo"},
			Line{Content: "foo"},
			Line{Content: "foo"},
			Line{Content: "foo"},
		},
		[]Line{
			Line{Content: "foo"},
			Line{Content: "foo"},
			Line{Content: "foo"},
			Line{Content: "foo"},
			Line{Content: "foo"},
		},
		[]Line{},
	}

	for i := 0; i < len(ccs); i++ {
		time.Sleep(timeout)
		t.Logf("start test %d, timeout expected: %s",
			i,
			strconv.FormatBool(shouldTimeout[i]))

		for _, line := range preLines[i] {
			in <- line
			t.Logf("pre: %s", line.Content)
		}

		time.Sleep(timeout) // avoid overlap with pre-lines

		// send condition
		start := time.Now()

		cc <- ccs[i]
		t.Logf("condition: %s, %d, %s",
			ccs[i].Condition.AcceptPattern.String(),
			ccs[i].Condition.Count,
			ccs[i].Condition.Timeout,
		)

		for _, line := range postLines[i] {
			in <- line
			t.Logf("post: %s", line.Content)
		}

		select {
		case <-ccs[i].Satisfied:
			duration := time.Since(start)
			t.Logf("finished test %d, timeout expected: %s, duration: %s",
				i,
				strconv.FormatBool(shouldTimeout[i]),
				duration)
			// we assume it passed the check if it returns within the
			// threshold time, which is less than the timeout
			assert.Equal(t, shouldTimeout[i], duration >= timeout)
			t.Logf("----------------------------------------------")

		case <-time.After(time.Second):
			t.Error("condition has not timed out in time")
		}
	}

}

func TestPlay(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	closed := make(chan struct{})

	conditionTimeout := 100 * time.Millisecond //must match total timeout(s) of condition(s) in play

	// there is one condition with a timeout of 100ms
	// we rely on this to check response to condition
	play := `{"some":"msg"}
# Non echo comment
#- non echo comment
#+ echo comment
[0.1s] {"an":"other"}
[] {"yet an":"other"}
<'^foo\s*',5,100ms> {"send":"foos"}
[0.1] {"not":"sent","bad":"delay format"}
<'^foo\s*',,10s> {"send":"foos"}
<'^foo\s*',5,> {"send":"foos"}
|+> [a-h]
|accept> [R-Z]
|->[0-9]
|deny>  [#!&%]
|reset>
|A> [a-h]
|D> [0-9]
|r> 
|X>
|a> ^\/(?!\/)(.*?)
goodbye
#+ finished example.play
` //place ` on separate line to ensure newline on last line with content

	sent := []string{
		`{"some":"msg"}`,
		`{"an":"other"}`,
		`{"yet an":"other"}`,
		`{"send":"foos"}`,
		`goodbye`,
	}

	actions := []FilterAction{
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
		FilterAction{
			Verb: Reset,
		},
		FilterAction{
			Verb:    Accept,
			Pattern: regexp.MustCompile(`[a-h]`),
		},
		FilterAction{
			Verb:    Deny,
			Pattern: regexp.MustCompile(`[0-9]`),
		},
		FilterAction{
			Verb: Reset,
		},
	}

	comments := []Line{
		Line{Content: `echo comment`},
		Line{Content: `finished example.play`},
	}

	n := strings.Count(play, "\n") + 1 //add one in case closing string quote on wrong line

	pout := make(chan interface{}, n)

	go func() {
		// won't complete until last value read from channel
		err := ParseByLine(strings.NewReader(play), pout)
		assert.NoError(t, err)
		t.Logf("ParseByLine completed")
	}()

	lines := []interface{}{}

	for l := range pout {
		lines = append(lines, l)
	}

	s := make(chan string, n)
	a := make(chan FilterAction, n)
	c := make(chan ConditionCheck, n)
	w := make(chan Line, n)
	//func Play(ctx context.Context, closed chan struct{}, lines []interface{}, a chan FilterAction, s chan string, c chan ConditionCheck) {

	// immediately satisfy any conditions - we'll check delays on this later ...
	go func() {
		defer func() {
			t.Logf("stopped first mock condition checker")
		}()
		t.Logf("starting first mock condition checker")
		for {
			select {
			case <-ctx.Done():
				return
			case <-closed:
				return
			case cc := <-c:
				t.Logf("Condition received: %s,%d,%s",
					cc.Condition.AcceptPattern.String(),
					cc.Condition.Count,
					cc.Condition.Timeout)
				t.Logf("satisfying condition immediately")
				close(cc.Satisfied)
			}

		}
	}()

	start := time.Now()
	var durationNoCondition time.Duration

	go Play(ctx, closed, lines, a, s, c, w)

	select {
	case <-closed:
		durationNoCondition = time.Since(start) //record natural running speed without conditions
	case <-time.After(time.Second):
		t.Error("Play did not finish in time")
	}

	// drain channel s

	// note if do not drain these channels, the test will hang
	// because we only buffered enough for one run of Play at a time

	for _, exp := range sent {
		select {
		case actual := <-s:
			assert.Equal(t, exp, actual)
		case <-time.After(time.Millisecond):
			t.Errorf("did not get message,expected %s", exp)
		}
	}

	// drain channel a
	for _, exp := range actions {
		select {
		case actual := <-a:
			t.Logf("action %v", actual)
			assert.Equal(t, exp, actual)
		case <-time.After(time.Millisecond):
			t.Errorf("did not get action,expected %v", exp)
		}
	}

	// drain channel w
	for _, exp := range comments {
		select {
		case actual := <-w:
			t.Logf("comment %v", actual.Content)
			// ignore the time field, as we cannot control it
			assert.Equal(t, exp.Content, actual.Content)
		case <-time.After(time.Millisecond):
			t.Errorf("did not get comment,expected %v", exp)
		}
	}

	// now we add the delay on the condition(s)
	go func() {
		defer func() {
			t.Logf("stopped first mock condition checker")
		}()
		t.Logf("starting second mock condition checker")
		for {
			select {
			case <-ctx.Done():
				return
			case <-closed:
				return
			case cc := <-c:
				t.Logf("Condition received: %s,%d,%s",
					cc.Condition.AcceptPattern.String(),
					cc.Condition.Count,
					cc.Condition.Timeout)
				<-time.After(cc.Condition.Timeout)
				t.Logf("satisfying condition after %s", cc.Condition.Timeout)
				close(cc.Satisfied)
			}

		}
	}()

	start = time.Now()

	closed = make(chan struct{})
	go Play(ctx, closed, lines, a, s, c, w)

	select {
	case <-closed:
		duration := time.Since(start)
		t.Logf("duration: %s", duration)
		// add a tolerance for variability between runs due to factors outside control
		// of the test, e.g. CPU loading
		tolerance := 10 * time.Millisecond
		if (duration - durationNoCondition) <= (conditionTimeout - tolerance) {
			t.Errorf("Play did not respect delay on condition check; delay amount %s, expected %s",
				duration-durationNoCondition,
				conditionTimeout)
		}
	case <-time.After(time.Second):
		t.Error("Play did not finish in time")
	}
}

func TestParseByLine(t *testing.T) {

	// put closing quote on its own line to ensure test counts lines
	// correctly (else test will hang forever)
	s := `{"some":"msg"}
# Non echo comment
#- non echo comment
#+ echo comment
[0.1s] {"an":"other"}
[] {"an":"other"}
<'^foo\s*',5,0.3h1.5m0.1s> {"send":"foos"}
[0.1] {"an":"other"}
<'^foo\s*',,10s> {"send":"foos"}
<'^foo\s*',5,> {"send":"foos"}
|+> [a-h]
|accept> [R-Z]
|->[0-9]
|deny>  [#!&%]
|reset>
|A> [a-h]
|D> [0-9]
|r> 
|X>
|a> ^\/(?!\/)(.*?)
[10ms]
`

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	go func() {
		select {
		case <-ctx.Done():
		case <-time.After(5 * time.Second):
			// see definition of s (above) and assertion check on n (below)
			fmt.Printf("test hung - check formatting of multiline string")
		}
	}()

	n := strings.Count(s, "\n")
	assert.Equal(t, 21, n) // Update this after editing

	expected := make([]interface{}, n)
	expected[0] = Send{
		Msg: `{"some":"msg"}`,
	}
	expected[1] = Comment{
		Msg:  "Non echo comment",
		Echo: false,
	}

	expected[2] = Comment{
		Msg:  "non echo comment",
		Echo: false,
	}

	expected[3] = Comment{
		Msg:  "echo comment",
		Echo: true,
	}

	expected[4] = Send{
		Msg:   `{"an":"other"}`,
		Delay: time.Millisecond * 100,
	}

	expected[5] = Send{
		Msg: `{"an":"other"}`,
	}

	expected[6] = Send{
		Msg: `{"send":"foos"}`,
		Condition: Condition{
			AcceptPattern: *regexp.MustCompile(`^foo\s*`),
			Count:         5,
			Timeout:       time.Second*((19*60)+30) + time.Millisecond*100,
		},
	}

	expected[7] = Error{`unknown delay time format: [0.1] {"an":"other"}`}

	expected[8] = Error{`malformed condition command [<'^foo\s*',,10s> {"send":"foos"} '^foo\s*',,10s {"send":"foos"}]; second argument  should be integer, count of messages to await. Line was: <'^foo\s*',,10s> {"send":"foos"}`}

	expected[9] = Error{`malformed condition command [<'^foo\s*',5,> {"send":"foos"} '^foo\s*',5, {"send":"foos"}]; third argument  should be timeout duration in format like 10s or 1m. Yours could not be parsed because time: invalid duration "". Line was was <'^foo\s*',5,> {"send":"foos"}`}

	expected[10] = FilterAction{
		Verb:    Accept,
		Pattern: regexp.MustCompile(`[a-h]`),
	}

	expected[11] = FilterAction{
		Verb:    Accept,
		Pattern: regexp.MustCompile(`[R-Z]`),
	}

	expected[12] = FilterAction{
		Verb:    Deny,
		Pattern: regexp.MustCompile(`[0-9]`),
	}

	expected[13] = FilterAction{
		Verb:    Deny,
		Pattern: regexp.MustCompile(`[#!&%]`),
	}

	expected[14] = FilterAction{
		Verb: Reset,
	}

	expected[15] = expected[10]

	expected[16] = expected[12]

	expected[17] = expected[14]

	expected[18] = Error{`malformed filter command; first argument not one of [+,-,a,d,r,accept,deny,reset], but was X`}

	expected[19] = Error{`malformed filter command; last argument ^\/(?!\/)(.*?) should be regexp pattern, but did not compile because error parsing regexp: invalid or unsupported Perl syntax: ` + "`(?!`. " + `Line was |a> ^\/(?!\/)(.*?)`}

	expected[20] = Wait{
		Delay: 10 * time.Millisecond,
	}

	in := strings.NewReader(s)

	out := make(chan interface{}, n) // buffer >= lines in s to avoid hang

	go func() {
		// won't complete until last value read from channel
		err := ParseByLine(in, out)
		assert.NoError(t, err)
		t.Logf("ParseByLine completed")
	}()

	idx := 0
	for o := range out {
		assert.Equal(t, expected[idx], o)
		t.Logf("%d: %v\n", idx, o)
		idx++
	}
	t.Logf("Note that Error{} are being passed due to checking handling of deliberately malformed commands - these are not errors in the test")

	assert.Equal(t, n, idx)

}
