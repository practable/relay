package file

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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

	err := ParseByLine(strings.NewReader(play), pout)

	assert.NoError(t, err)

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
		durationNoCondition = time.Now().Sub(start) //record natural running speed without conditions
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
		duration := time.Now().Sub(start)
		t.Logf("duration: %s", duration)
		if duration-durationNoCondition < conditionTimeout {
			t.Error("Play did not respect delay on condition check")
		}
	case <-time.After(time.Second):
		t.Error("Play did not finish in time")
	}

	t.Log("todo Check that play writes echo messages to the log!")

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
	assert.Equal(t, 20, n) // Update this after editing

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
			AcceptPattern: *regexp.MustCompile("^foo\\s*"),
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

	in := strings.NewReader(s)

	out := make(chan interface{}, n) // buffer >= lines in s to avoid hang

	err := ParseByLine(in, out)

	assert.NoError(t, err)

	idx := 0
	for o := range out {
		assert.Equal(t, expected[idx], o)
		t.Logf("%d: %v\n", idx, o)
		idx++
	}
	t.Logf("Note that Error{} are being passed due to checking handling of deliberately malformed commands - these are not errors in the test")

	assert.Equal(t, n, idx)

}
