package mangoprovider

import (
	"regexp"
	"strings"
)

const (
	ChapterNumberIDName     = "chap_num"
	ChapterPartNumberIDName = "part_num"
	ChapterNameIDName       = "title"
	MangaQueryIDName        = "id"
)

var (
	// Old, keeping them for quickly switching just in case
	// https://regex101.com/r/ADDouB
	// ChapterNameRegex       = regexp.MustCompile(`(?mi)chapter\s*#?\s*\d+(\.\d+)?\s*:\s*(.*\S)\s*$`)
	// ChapterNameRegex       = regexp.MustCompile(`(?mi)^([a-z]*\.?)\s*#?\s*\d+(\.\d+)?\s*[:\-_.]?\s+(?P<title>.*\S)\s*$`)
	// ChapterNameRegex       = regexp.MustCompile(`(?mi)^([a-z]*\.?)\s*#?\s*(?P<chap_num>\d+(\.\d+)?)\D\s*([:\-_.,]?\s*part\s*(?P<part_num>\d+))?\s*[:\-_.,]?\s*(?P<title>.*\S)\s*$`)
	MangaQueryIDRegex       = regexp.MustCompile(`(?mi)\s*(m((anga)?[-_]?)?id)\s*:\s*(?P<id>.*\S)\s*$`)
	ChapterNumberRegex      = regexp.MustCompile(`(?m)(\d+\.\d+|\d+)`)
	ChapterNumberMPRegex    = regexp.MustCompile(`(?m)(\d+-\d+|\d+\.\d+|\d+)`)
	ChapterNameRegex        = regexp.MustCompile(`(?mi)^([a-z]*\.?)\s*#?\s*(?P<chap_num>\d+(\.\d+)?)\s*([:\-_.,]?\s*part\s*(?P<part_num>\d+))?\s*[:\-_.,]?\s*(?P<title>\S.*\S)\s*$`)
	ChapterNameExcludeRegex = regexp.MustCompile(`(?mi)^part\s*#?\s*\d+(\.\d+)?$`)
	NewlineCharactersRegex  = regexp.MustCompile(`\r?\n`)
	ImageExtensionRegex     = regexp.MustCompile(`^\.[a-zA-Z0-9][a-zA-Z0-9.]*[a-zA-Z0-9]$`)
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

// FloatIsInt is a helper function to see if the float value is actually an integer.
func FloatIsInt(number float32) bool {
	return number == float32(int(number))
}
