package manifest

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

var debug bool

func init() {

	debug = false

}

func TestExample(t *testing.T) {

	m0 := Example()

	_, err := yaml.Marshal(m0)

	assert.NoError(t, err)

	content, err := ioutil.ReadFile("testdata/example.yaml")
	assert.NoError(t, err)

	m1 := &Manifest{}
	err = yaml.Unmarshal(content, m1)
	assert.NoError(t, err)

	assert.Equal(t, m0, m1)

}

/*
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
*/
