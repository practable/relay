package shellbar

import (
	"bufio"
	"bytes"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func clientExists(topics *topicDirectory, client clientDetails) bool {

	topics.Lock()
	existingClients := topics.directory[client.topic]
	topics.Unlock()

	for _, existingClient := range existingClients {
		if client.name == existingClient.name {
			return true

		}
	}

	return false

}

func TestSlashify(t *testing.T) {

	if "/foo" != slashify("foo") {
		t.Errorf("Slashify not prefixing slash ")
	}
	if "//foo" == slashify("/foo") {
		t.Errorf("Slashify prefixing additional slash")
	}
	if "/foo" != slashify("/foo/") {
		t.Errorf("Slashify not removing trailing slash")
	}
	if "/foo" != slashify("foo/") {
		t.Errorf("Slashify not both removing trailing slash AND prefixing slash")
	}

	b := "foo/bar/rab/oof/"
	if "/foo/bar/rab/oof" != slashify(b) {
		t.Errorf("Slashify not coping with internal slashes %s -> %s", b, slashify(b))
	}

}

func suppressLog() {
	var ignore bytes.Buffer
	logignore := bufio.NewWriter(&ignore)
	log.SetOutput(logignore)
}

func displayLog() {
	log.SetOutput(os.Stdout)
}

func traceLog() {
	log.SetLevel(log.TraceLevel)
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true, DisableColors: true})
}

func debug(debug bool) func() {

	if debug {
		traceLog()
		return func() {}
	} else {
		suppressLog()
		return displayLog
	}
}

func TestGetPrefixFromPath(t *testing.T) {

	assert.Equal(t, "foo%20bar", GetPrefixFromPath("/foo%20bar/glum"))
	assert.Equal(t, "", GetPrefixFromPath("ooops/foo%20bar/glum"))

}
func TestGetTopicFromPath(t *testing.T) {

	assert.Equal(t, "glum", GetTopicFromPath("/foo%20bar/glum"))
	assert.Equal(t, "", GetTopicFromPath("ooops/foo%20bar/glum"))

}
