package util

import "encoding/json"

func Pretty(t interface{}) string {

	json, err := json.MarshalIndent(t, "", "\t")
	if err != nil {
		return ""
	}

	return string(json)
}
