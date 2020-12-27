package crossbar

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

func TestGetConnectionTypeFromPath(t *testing.T) {

	assert.Equal(t, "connectionType", getConnectionTypeFromPath("/connectionType/sessionID"))
	assert.Equal(t, "", getConnectionTypeFromPath("NoLeadingSlash/A/B/C"))
	assert.Equal(t, "foo%20bar", getConnectionTypeFromPath("/foo%20bar/glum"))
	assert.Equal(t, "", getConnectionTypeFromPath("ooops/foo%20bar/glum"))

}
func TestGetTopicFromPath(t *testing.T) {

	assert.Equal(t, "sessionID", getTopicFromPath("/connectionType/sessionID"))
	assert.Equal(t, "", getTopicFromPath("NoLeadingSlash/A/B/C"))
	assert.Equal(t, "session%20ID/connection%20ID", getTopicFromPath("/connectionType/session%20ID/connection%20ID"))
	assert.Equal(t, "sessionID/connectionID", getTopicFromPath("/connectionType/sessionID/connectionID?QueryParams=Something"))
	assert.Equal(t, "sessionID/connectionID", getTopicFromPath("/connectionType/sessionID/connectionID?QueryParams=Something&SomeThing=Else"))
}

func TestGetSessionIDFromPath(t *testing.T) {

	assert.Equal(t, "sessionID", getSessionIDFromPath("/connectionType/sessionID"))
	assert.Equal(t, "", getSessionIDFromPath("NoLeadingSlash/A/B/C"))
	assert.Equal(t, "session%20ID", getSessionIDFromPath("/connectionType/session%20ID/connection%20ID"))
	assert.Equal(t, "sessionID", getSessionIDFromPath("/connectionType/sessionID/connectionID?QueryParams=Something"))
	assert.Equal(t, "sessionID", getSessionIDFromPath("/connectionType/sessionID/connectionID?QueryParams=Something&SomeThing=Else"))
}

func TestGetConnectionIDFromPath(t *testing.T) {

	assert.Equal(t, "", getConnectionIDFromPath("/connectionType/sessionID"))
	assert.Equal(t, "", getConnectionIDFromPath("NoLeadingSlash/A/B/C"))
	assert.Equal(t, "connection%20ID", getConnectionIDFromPath("/connectionType/session%20ID/connection%20ID  "))
	assert.Equal(t, "connectionID", getConnectionIDFromPath("/connectionType/sessionID/connectionID?QueryParams=Something"))
	assert.Equal(t, "connectionID", getConnectionIDFromPath("/connectionType/sessionID/connectionID?QueryParams=Something&SomeThing=Else"))

}
