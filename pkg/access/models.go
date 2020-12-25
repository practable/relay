package access

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/timdrysdale/relay/pkg/ttlcode"
)

type Options struct {
	disableAuth bool //not stable part of API

}

func NewOptions(cs *ttlcode.CodeStore) *Options {
	return &Options{}
}

// GetScopes returns a map of the scopes allowed for each
// path, with the path prefix as map key
// e.g. https://relay-access.yourdomain.io/session/8410928349108230498
// has the path prefix session
func getScopesAll() map[string][]string {
	scopes := make(map[string][]string)
	scopes["session"] = []string{"read", "write"}
	scopes["shell"] = []string{"host", "client"}
	scopes["stats"] = []string{"read"}
	return scopes
}

// GetScopes returns the scopes allowed for paths with the prefix
func getScopesForPrefix(prefix string) []string {

	allScopes := getScopesAll()

	scopes, ok := allScopes[prefix]

	if !ok {
		return []string{}
	}

	return scopes
}

// isValidScopesFor returns true if one or more of the supplied
// scopes is an allowed scope for paths with the prefix
func isValidScopesFor(prefix string, scopes []string) bool {

	allowedScopes := getScopesForPrefix(prefix)

	for _, scope := range scopes {
		for _, allowed := range allowedScopes {
			if scope == allowed {
				return true
			}
		}
	}

	return false

}

func checkScopesForPath(path string, scopes []string) error {

	prefix := getPrefixFromPath(path)

	allowedScopes := getScopesForPrefix(prefix)

	if !isValidScopesFor(prefix, scopes) {
		return fmt.Errorf("path %s has prefix %s allowing scopes %v but none found in %v", path, prefix, allowedScopes, scopes)
	}

	return nil
}

func getPrefixFromPath(path string) string {

	re := regexp.MustCompile("^\\/([\\w\\%-]*)\\/")

	matches := re.FindAllString(path, 2)

	if len(matches) < 2 {
		return ""
	}

	// matches[0] = "/{prefix}/"
	// matches[1] = "{prefix}"
	return matches[1]
}

func TestGetPrefixFromPath(t *testing.T) {

	assert.Equal(t, "foo%20bar", getPrefixFromPath("/foo%20bar/glum"))
	assert.Equal(t, "", getPrefixFromPath("ooops/foo%20bar/glum"))

}
