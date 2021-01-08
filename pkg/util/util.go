package util

import (
	"encoding/json"
	"sort"
)

func Pretty(t interface{}) string {

	json, err := json.MarshalIndent(t, "", "\t")
	if err != nil {
		return ""
	}

	return string(json)
}
func Compact(t interface{}) string {

	json, err := json.Marshal(t)
	if err != nil {
		return ""
	}

	return string(json)
}

// https://stackoverflow.com/questions/52395494/best-way-to-check-if-two-arrays-have-the-same-members  by RayfenWindspear - This is not correct ....
// DO NOT USE
func unorderedEqual(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	exists := make(map[string]bool)
	for _, value := range first {
		exists[value] = true
	}
	for _, value := range second {
		if !exists[value] {
			return false
		}
	}
	return true
}

// run it twice and it becomes correct...
func DoubleUnorderedEqual(a, b []string) bool {
	return unorderedEqual(a, b) && unorderedEqual(b, a)
}

// putting Husain's code into a function...
// BTW this is 10x faster for a few dozen entries in each list
func SortCompare(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	sort.Strings(a)
	sort.Strings(b)

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
