package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
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

// ParseCreationDate parses datestring using iso8601. If the year is "????", replace it with year 0000
func ParseCreationDate(datestring string) (time.Time, error) {
	parsedDate, err := iso8601.ParseString(
		strings.ReplaceAll(
			strings.Replace(datestring, "????", "0000", 1), "?", "1",
		),
	)
	return parsedDate, err
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

func printfln(text string, a ...interface{}) {
	fmt.Printf(text+"\n", a...)
}

func printerr(explanation string, err error) {
	printfln(explanation+": %s", err)
}
