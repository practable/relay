package shellaccess

import (
	"regexp"
)

func getPrefixFromPath(path string) string {

	re := regexp.MustCompile(`^\/([\w\%-]*)\/`)

	matches := re.FindStringSubmatch(path)
	if len(matches) < 2 {
		return ""
	}

	// matches[0] = "/{prefix}/"
	// matches[1] = "{prefix}"
	return matches[1]

}
