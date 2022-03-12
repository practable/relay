package file

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseByLine(t *testing.T) {

	s := `{"some":"msg"}
# Non echo comment
#- non echo comment 
#+ echo comment
[0.1s] {"an":"other"}
[] {"an":"other"}
<'^foo\s*',5,0.3h1.5m0.1s> {"send":"foos"}

`

	in := strings.NewReader(s)

	out := make(chan interface{}, 10) // buffer >= lines in s to avoid hang

	err := ParseByLine(in, out)

	assert.NoError(t, err)

	for o := range out {
		fmt.Printf("%v\n", o)
	}

}
