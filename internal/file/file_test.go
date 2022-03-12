package file

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	flag.Parse() // needed to use testing.Verbose https://github.com/golang/go/issues/9825

	if testing.Verbose() {
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true, DisableColors: true})
		defer log.SetOutput(os.Stdout)

	} else {
		var ignore bytes.Buffer
		logignore := bufio.NewWriter(&ignore)
		log.SetOutput(logignore)
	}

	exitVal := m.Run()

	os.Exit(exitVal)
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
	assert.Equal(t, 10, n) // Update this after editing

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
			Filter:  *regexp.MustCompile("^foo\\s*"),
			Count:   5,
			Timeout: time.Second*((19*60)+30) + time.Millisecond*100,
		},
	}

	expected[7] = Error{`unknown delay time format: [0.1] {"an":"other"}`}

	expected[8] = Error{`malformed condition command [<'^foo\s*',,10s> {"send":"foos"} '^foo\s*',,10s {"send":"foos"}]; second argument  should be integer, count of messages to await. Line was: <'^foo\s*',,10s> {"send":"foos"}`}

	expected[9] = Error{`malformed condition command [<'^foo\s*',5,> {"send":"foos"} '^foo\s*',5, {"send":"foos"}]; third argument  should be timeout duration in format like 10s or 1m. Yours could not be parsed because time: invalid duration "". Line was was <'^foo\s*',5,> {"send":"foos"}`}

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
