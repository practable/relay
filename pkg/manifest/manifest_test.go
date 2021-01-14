package manifest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestGetManifest(t *testing.T) {

	ps := Example()

	fmt.Println(ps)

	m := GetManifest(ps)

	fmt.Println(m)

	buf, err := yaml.Marshal(m)

	assert.NoError(t, err)

	fmt.Println(string(buf))

}
