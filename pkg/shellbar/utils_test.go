package shellbar

import (
	"bufio"
	"bytes"
	"os"
	"testing"

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

func TestGetConnectionTypeFromPath(t *testing.T) {

	assert.Equal(t, "connectionType", getConnectionTypeFromPath("/connectionType/shellID"))
	assert.Equal(t, "", getConnectionTypeFromPath("NoLeadingSlash/A/B/C"))
	assert.Equal(t, "foo%20bar", getConnectionTypeFromPath("/foo%20bar/glum"))
	assert.Equal(t, "", getConnectionTypeFromPath("ooops/foo%20bar/glum"))
}

func TestGetHostTopicFromUniqueTopic(t *testing.T) {

	assert.Equal(t, "shellID", getHostTopicFromUniqueTopic("shellID"))
	assert.Equal(t, "NoLeadingSlash", getHostTopicFromUniqueTopic("NoLeadingSlash/A/B/C"))
	assert.Equal(t, "", getHostTopicFromUniqueTopic("/foo%20bar/glum"))
	assert.Equal(t, "ooops", getHostTopicFromUniqueTopic("ooops/foo%20bar/glum"))
}

func TestGetTopicFromPath(t *testing.T) {

	assert.Equal(t, "shellID", getTopicFromPath("/connectionType/shellID"))
	assert.Equal(t, "", getTopicFromPath("NoLeadingSlash/A/B/C"))
	assert.Equal(t, "shell%20ID/connection%20ID", getTopicFromPath("/connectionType/shell%20ID/connection%20ID"))
	assert.Equal(t, "shellID/connectionID", getTopicFromPath("/connectionType/shellID/connectionID?QueryParams=Something"))
	assert.Equal(t, "shellID/connectionID", getTopicFromPath("/connectionType/shellID/connectionID?QueryParams=Something&SomeThing=Else"))
}

func TestGetShellIDFromPath(t *testing.T) {

	assert.Equal(t, "shellID", getShellIDFromPath("/connectionType/shellID"))
	assert.Equal(t, "", getShellIDFromPath("NoLeadingSlash/A/B/C"))
	assert.Equal(t, "shell%20ID", getShellIDFromPath("/connectionType/shell%20ID/connection%20ID"))
	assert.Equal(t, "shellID", getShellIDFromPath("/connectionType/shellID/connectionID?QueryParams=Something"))
	assert.Equal(t, "shellID", getShellIDFromPath("/connectionType/shellID/connectionID?QueryParams=Something&SomeThing=Else"))
}

func TestGetConnectionIDFromPath(t *testing.T) {

	assert.Equal(t, "", getConnectionIDFromPath("/connectionType/shellID"))
	assert.Equal(t, "", getConnectionIDFromPath("NoLeadingSlash/A/B/C"))
	assert.Equal(t, "connection%20ID", getConnectionIDFromPath("/connectionType/shell%20ID/connection%20ID  "))
	assert.Equal(t, "connectionID", getConnectionIDFromPath("/connectionType/shellID/connectionID?QueryParams=Something"))
	assert.Equal(t, "connectionID", getConnectionIDFromPath("/connectionType/shellID/connectionID?QueryParams=Something&SomeThing=Else"))

}
