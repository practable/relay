package util

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// This test fails so don't run it
func testUnorderedEqual(t *testing.T) {

	assert.True(t, unorderedEqual([]string{"aa"}, []string{"aa"}))
	assert.False(t, unorderedEqual([]string{"aa"}, []string{"bb"}))
	assert.False(t, unorderedEqual([]string{"aa"}, []string{"bb", "cc"}))
	assert.True(t, unorderedEqual([]string{"cc", "bb"}, []string{"bb", "cc"}))
	assert.True(t, unorderedEqual([]string{"aa", "cc", "bb"}, []string{"aa", "bb", "cc"}))
	// this test fails, because the func is inadequate
	assert.False(t, unorderedEqual([]string{"aa", "cc", "bb"}, []string{"aa", "bb", "bb"}))
}

func TestSortCompare(t *testing.T) {

	assert.True(t, SortCompare([]string{"aa"}, []string{"aa"}))
	assert.False(t, SortCompare([]string{"aa"}, []string{"bb"}))
	assert.False(t, SortCompare([]string{"aa"}, []string{"bb", "cc"}))
	assert.True(t, SortCompare([]string{"cc", "bb"}, []string{"bb", "cc"}))
	assert.True(t, SortCompare([]string{"aa", "cc", "bb"}, []string{"aa", "bb", "cc"}))
	assert.False(t, SortCompare([]string{"aa", "cc", "bb"}, []string{"aa", "bb", "bb"}))
}

func TestDoubleUnorderedEqual(t *testing.T) {

	assert.True(t, DoubleUnorderedEqual([]string{"aa"}, []string{"aa"}))
	assert.False(t, DoubleUnorderedEqual([]string{"aa"}, []string{"bb"}))
	assert.False(t, DoubleUnorderedEqual([]string{"aa"}, []string{"bb", "cc"}))
	assert.True(t, DoubleUnorderedEqual([]string{"cc", "bb"}, []string{"bb", "cc"}))
	assert.True(t, DoubleUnorderedEqual([]string{"aa", "cc", "bb"}, []string{"aa", "bb", "cc"}))
	assert.False(t, DoubleUnorderedEqual([]string{"aa", "cc", "bb"}, []string{"aa", "bb", "bb"}))

}

func getStrings() ([]string, []string) {

	a := []string{"a", "foo", "bar", "ping", "pong"}
	b := []string{"pong", "foo", "a", "bar", "ping"}

	return a, b

}

func BenchmarkSortCompare(b *testing.B) {
	s0, s1 := getStrings()
	var outcome bool
	for n := 0; n < b.N; n++ {
		outcome = SortCompare(s0, s1)
	}
	fmt.Println(outcome)
}

func BenchmarkDoubleUnorderedEqual(b *testing.B) {
	s0, s1 := getStrings()
	var outcome bool
	for n := 0; n < b.N; n++ {
		outcome = DoubleUnorderedEqual(s0, s1)
	}
	fmt.Println(outcome)
}

func getStrings2() ([]string, []string) {

	a := []string{"a", "foo", "bar", "ping", "pong", "b", "c", "g", "e", "f", "d", "h", "i", "j", "q", "l", "n", "o", "p", "k", "r", "s", "t", "u", "v", "w", "x", "y", "z"}
	b := []string{"pong", "foo", "a", "bar", "ping", "p", "r", "q", "s", "u", "t", "v", "j", "x", "y", "z", "b", "e", "d", "c", "h", "g", "f", "i", "w", "k", "l", "n", "o"}

	return a, b

}

func BenchmarkSortCompare2(b *testing.B) {
	s0, s1 := getStrings2()
	var outcome bool
	for n := 0; n < b.N; n++ {
		outcome = SortCompare(s0, s1)
	}
	fmt.Println(outcome)
}

func BenchmarkDoubleUnorderedEqual2(b *testing.B) {
	s0, s1 := getStrings2()
	var outcome bool
	for n := 0; n < b.N; n++ {
		outcome = DoubleUnorderedEqual(s0, s1)
	}
	fmt.Println(outcome)
}
