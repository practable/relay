package cmd

import (
	"fmt"
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
