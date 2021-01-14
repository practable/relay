package manifest

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/r3labs/diff"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

var debug bool

func init() {

	debug = false

}

func TestGetManifest(t *testing.T) {

	ps := Example()
	assert.Equal(t, 2, len(ps.Pools), "incorrect pool count")
	assert.Equal(t, 2, len(ps.Groups), "incorrect group count")

	if debug {
		pretty, err := json.MarshalIndent(ps, "", "\t")
		assert.NoError(t, err)
		fmt.Println(string(pretty))
	}

	m := GetManifest(ps)

	buf, err := yaml.Marshal(m)

	assert.NoError(t, err)

	if debug {
		fmt.Println(string(buf))
	}

	ps2 := m.GetPoolStore()

	if false {
		pretty, err := json.MarshalIndent(ps2, "", "\t")
		assert.NoError(t, err)
		fmt.Println(string(pretty))
	}

	changeLog, err := diff.Diff(*ps, *ps2)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(ps2.Pools), "incorrect pool count")
	assert.Equal(t, 2, len(ps2.Groups), "incorrect group count")

	if true {
		pretty, err := json.MarshalIndent(changeLog, "", "\t")
		assert.NoError(t, err)
		fmt.Println(string(pretty))
	}

}
