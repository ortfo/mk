package ortfomk

import (
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/relvacode/iso8601"
)

// FindInArrayLax checks if needle is in haystack, ignoring case and whitespace around values
func FindInArrayLax(haystack []string, needle string) (string, error) {
	for _, element := range haystack {
		if strings.TrimSpace(strings.ToLower(element)) == needle {
			return needle, nil
		}
	}
	return "", errors.New("not found")
}

// IsColorHexstring determines if the given string is a valid 6-character hexstring
func IsColorHexstring(s string) bool {
	_, err := hex.DecodeString(s)
	if err != nil {
		return false
	}
	return len(s) == 6
}

// AddOctothorpeIfNeeded adds a leading "#" to colorValue if it's a valid color hexstring
func AddOctothorpeIfNeeded(colorValue string) string {
	if IsColorHexstring(colorValue) {
		return "#" + colorValue
	}
	return colorValue
}

// SummarizeString summarizes s to at most targetLength characters
// TODO: do not cut in between of a word/punctuation mark, etc.
func SummarizeString(s string, targetLength uint32) string {
	var runesCount uint32
	for index := range s {
		runesCount++
		if runesCount > targetLength {
			return s[:index] + "…"
		}
	}
	return s
}

// ParseCreationDate parses datestring using iso8601. If the year is "????", replace it with year 9999
func ParseCreationDate(datestring string) (time.Time, error) {
	return iso8601.ParseString(
		strings.ReplaceAll(
			strings.Replace(datestring, "????", "9999", 1), "?", "1",
		),
	)
}

// StringsLooselyMatch checks if s1 is equal to any of sn, but case-insensitively.
func StringsLooselyMatch(s1 string, sn ...string) bool {
	for _, s2 := range sn {
		if strings.EqualFold(s1, s2) {
			return true
		}
	}
	return false
}

// func printfln(text string, a ...interface{}) {
// 	fmt.Printf(text+"\n", a...)
// }

// lcm returns the least common multiple of all the provided integers
func lcm(integers ...int) int {
	if len(integers) < 2 {
		return integers[0]
	}
	var greater int
	// choose the greater number
	if integers[0] > integers[1] {
		greater = integers[0]
	} else {
		greater = integers[1]
	}

	for {
		if (greater%integers[0] == 0) && (greater%integers[1] == 0) {
			break
		}
		greater += 1
	}
	if len(integers) == 2 {
		return greater
	}
	return lcm(append(integers[2:], greater)...)
}

const MaxInt = int(^uint(0) >> 1)

//
// Testing utilities (only used in *_test.go)
//

type testData struct {
	have interface{}
	want interface{}
}

type test struct {
	testData

	*testing.T
}

func (td test) Do(iter int) {
	if td.have != td.want {
		td.Errorf("failed test data #%d:\n\twant: %v\n\thave: %v", iter, td.want, td.have)
	}
}

func runTestTable(t *testing.T, tests []testData) {
	for i, testData := range tests {
		test{testData, t}.Do(i)
	}
}

func assertPanic(t *testing.T, f func()) {
	defer func() {
		if recover() == nil {
			t.Errorf("did not panic")
		}
	}()
	f()
}

func keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// filepathStem returns the base of a path with the extension removed
func filepathStem(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func deduplicate[T comparable](items []T) []T {
	noDuplicates := make([]T, 0, len(items))
outer:
	for _, item := range items {
		for _, i := range noDuplicates {
			if i == item {
				continue outer
			}
		}
		noDuplicates = append(noDuplicates, item)
	}
	return noDuplicates
}

func excluding[T comparable](items []T, excluded ...T) []T {
	withoutRemoved := make([]T, 0, len(items))
outer:
	for _, item := range items {
		for _, excludedItem := range excluded {
			if item == excludedItem {
				continue outer
			}
		}
		withoutRemoved = append(withoutRemoved, item)
	}
	return withoutRemoved
}
