package shellbar

import (
	"fmt"
	"regexp"
	"strings"
)

func filterClients(clients []clientDetails, filter clientDetails) []clientDetails {
	filteredClients := clients[:0]
	for _, client := range clients {
		if client.name != filter.name {
			filteredClients = append(filteredClients, client)
		}
	}
	return filteredClients
}

func slashify(path string) string {

	//remove trailing slash (that's for directories)
	path = strings.TrimSuffix(path, "/")

	//ensure leading slash without needing it in config
	path = strings.TrimPrefix(path, "/")
	path = fmt.Sprintf("/%s", path)

	return path

}

func GetPrefixFromPath(path string) string {

	re := regexp.MustCompile("^\\/([\\w\\%-]*)\\/")

	matches := re.FindStringSubmatch(path)

	if len(matches) < 2 {
		return ""
	}

	// matches[0] = "/{prefix}/"
	// matches[1] = "{prefix}"
	return matches[1]
}

func GetTopicFromPath(path string) string {

	re := regexp.MustCompile("^\\/[\\w\\%-]*\\/([\\w\\%-]*)")

	matches := re.FindStringSubmatch(path)

	if len(matches) < 2 {
		return ""
	}

	// matches[0] = "/{prefix}/{topic}"
	// matches[1] = "{topic}"
	return matches[1]
}
