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
[0.1s] {"an":"other"}`

	in := strings.NewReader(s)

	out := make(chan interface{}, 5) //make it buffered to avoid blocking

	err := ParseByLine(in, out)

	assert.NoError(t, err)

	for o := range out {
		fmt.Printf("%+v\n", o)
	}

}
