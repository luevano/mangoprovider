package mangoprovider

import (
	"regexp"
	"strings"
)

var (
	ChapterNumberRegex     = regexp.MustCompile(`(?m)(\d+\.\d+|\d+)`)
	ChapterNameRegex       = regexp.MustCompile(`(?mi)chapter\s*#?\s*\d+(\.\d+)?\s*:\s*(.*\S)\s*$`)
	NewlineCharactersRegex = regexp.MustCompile(`\r?\n`)
)

// Returns the string with single spaces. E.g. "    " -> " "
func standardizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// Get the string with all whitespace standardized.
func CleanString(s string) string {
	return standardizeSpaces(NewlineCharactersRegex.ReplaceAllString(s, " "))
}
