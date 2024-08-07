package mangoprovider

import (
	"regexp"
	"strconv"
	"strings"
)

const (
	ChapterNumberIDName     = "chap_num"
	ChapterPartNumberIDName = "part_num"
	ChapterNameIDName       = "title"
	MangaQueryIDName        = "id"
)

const chNumRe = `\d+([\.-]\d+)?`

var (
	MangaQueryIDRegex       = regexp.MustCompile(`(?mi)\s*(m((anga)?[-_]?)?id)\s*:\s*(?P<id>.*\S)\s*$`)
	ChapterNumberRegex      = regexp.MustCompile(chNumRe)
	ChapterNameRegex        = regexp.MustCompile(`(?mi)^([a-z]*\.?)\s*#?\s*(?P<chap_num>` + chNumRe + `)\s*([:\-_.,]?\s*part\s*(?P<part_num>` + chNumRe + `))?\s*[:\-_.,]?\s+(?P<title>\S.*\S)\s*$`) // https://regex101.com/r/ADDouB
	ChapterNameExcludeRegex = regexp.MustCompile(`(?mi)^part\s*#?\s*` + chNumRe + `$`)
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

// Get the float number with all of the insignificant digits removed.
//
// For example, "001.500" becomes "1.5".
func FormattedFloat(n float32) string {
	return strconv.FormatFloat(float64(n), 'f', -1, 32)
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
