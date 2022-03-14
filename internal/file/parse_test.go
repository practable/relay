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

	assert.Equal(t, n, idx)

}
