package mangoprovider

import (
	"regexp"
	"strings"
)

const MangaQueryIDName = "id"

var (
	MangaQueryIDRegex      = regexp.MustCompile(`(?i)\s*(m((anga)?[-_]?)?id)\s*:\s*(?P<id>.*\S)\s*$`)
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

// Translate regex named groups to a map.
//
// https://stackoverflow.com/a/53587770
func ReNamedGroups(pattern *regexp.Regexp, str string) map[string]string {
	groups := make(map[string]string)
	match := pattern.FindStringSubmatch(str)
	for i, value := range match {
		name := pattern.SubexpNames()[i]
		if name != "" {
			groups[name] = value
		}
	}
	return groups
}
