package mangoprovider

import (
	"regexp"
	"strconv"
	"strings"
)

const (
	VolumeNumberID  = "vol_num"
	ChapterNumberID = "chap_num"
	PartNumberID    = "part_num"
	ChapterTitleID  = "title"
	MangaQueryID    = "id"
)

const (
	numRe      = `\d+([\.-]\d+)?`
	sepRe      = `[:\-_.,]?`
	numSepRe   = `\s*#?\s*`
	volNumRe   = `([a-z]*\.?)` + numSepRe + `(?P<` + VolumeNumberID + `>` + numRe + `)`
	chapNumRe  = `([a-z]*\.?)` + numSepRe + `(?P<` + ChapterNumberID + `>` + numRe + `)`
	partNumRe  = `(part)` + numSepRe + `(?P<` + PartNumberID + `>` + numRe + `)`
	chapNameRe = `(?P<` + ChapterTitleID + `>\S.*\S)`
)

var (
	MangaQueryIDRegex       = regexp.MustCompile(`(?mi)\s*(m((anga)?[-_]?)?id)\s*:\s*(?P<` + MangaQueryID + `>.*\S)\s*$`)
	ChapterNumberRegex      = regexp.MustCompile(numRe)
	ChapterNameRegex        = regexp.MustCompile(`(?mi)^(` + volNumRe + `)?\s*` + chapNumRe + `\s*(` + sepRe + `\s*` + partNumRe + `)?\s*` + sepRe + `\s+` + chapNameRe + `\s*$`) // https://regex101.com/r/ADDouB
	ChapterNameExcludeRegex = regexp.MustCompile(`(?mi)^` + partNumRe + `$`)
	NewlineCharactersRegex  = regexp.MustCompile(`\r?\n`)
	ImageExtensionRegex     = regexp.MustCompile(`^\.[a-zA-Z0-9][a-zA-Z0-9.]*[a-zA-Z0-9]$`)
)

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

// FloatIsInt is a helper function to see if the float value is actually an integer.
func FloatIsInt(number float32) bool {
	return number == float32(int(number))
}
