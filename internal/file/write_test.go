package file

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatLine(t *testing.T) {

	l := Line{
		Time:    time.Date(2022, 3, 13, 19, 15, 32, 15000000, time.UTC),
		Content: `There was a "bright" spark`,
	}

	s := FormatLine(l)

	exp := `[2022-03-13T19:15:32.015Z] There was a "bright" spark
` // ` has to be on this following line to include a genuine \n

	assert.Equal(t, exp, s)

}

func TestWrite(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	var b bytes.Buffer

	in := make(chan Line, 5) //buffer >= lines sent to avoid hang

	l0 := Line{
		Time:    time.Date(2022, 3, 13, 19, 15, 32, 15000000, time.UTC),
		Content: `There was a "bright" spark`,
	}

	l1 := Line{
		Time:    time.Date(2022, 3, 13, 19, 15, 32, 15200000, time.UTC),
		Content: `And lo the fire was lit.`,
	}

	exp := `[2022-03-13T19:15:32.015Z] There was a "bright" spark
[2022-03-13T19:15:32.0152Z] And lo the fire was lit.
`

	in <- l0
	in <- l1

	close(in)

	Write(ctx, in, &b) //run not as goro to make sure it writes to buffer before check
	assert.Equal(t, exp, b.String())

}
