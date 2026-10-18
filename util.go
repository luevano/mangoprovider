package mangoprovider

import (
	"fmt"
	"math"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/luevano/libmangal/metadata"
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
	volNumRe   = `(v[a-z]*\.?)` + numSepRe + `(?P<` + VolumeNumberID + `>` + numRe + `)`
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

func ParseChapterTitle(s string) string {
	// Need to normalize the spaces, some weird unicode spaces are not matched with regex
	title := strings.TrimSpace(CleanString(s))

	// Try to get the name without prefix "Chapter 123:" or similar
	matchGroups := ReNamedGroups(ChapterNameRegex, title)
	titleTemp := strings.TrimSpace(matchGroups[ChapterTitleID])
	if titleTemp != "" {
		// Check that the resulting title is not just "Part 123",
		// as it probably is part of the whole title and we'll like to keep
		// the prefix
		// This happens with Spy x Family: "Mission X Part Y" for example
		if !ChapterNameExcludeRegex.MatchString(titleTemp) {
			title = titleTemp
			// Add the "part number" at the end of the title if existent
			partNum := strings.TrimSpace(matchGroups[PartNumberID])
			if partNum != "" {
				title = fmt.Sprintf("%s, Part %s", title, partNum)
			}
		}
	}
	return title
}

// Returns the string with single spaces. E.g. "    " -> " "
func standardizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// Get the string with all whitespace standardized.
func CleanString(s string) string {
	return standardizeSpaces(NewlineCharactersRegex.ReplaceAllString(strings.Replace(s, "\t", " ", -1), " "))
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

// ParseChapterNumber returns the chapter number in the title, ok is false if there's none.
func ParseChapterNumber(title string) (number float32, ok bool) {
	match := strings.TrimSpace(ReNamedGroups(ChapterNameRegex, title)[ChapterNumberID])
	// if the groups matching fail, it probably only contains the chapter number
	if match == "" {
		match = ChapterNumberRegex.FindString(title)
	}
	if match == "" {
		return 0, false
	}
	// some sites use "-" for decimals, e.g. "12-5"
	match = strings.Replace(match, "-", ".", 1)
	parsed, err := strconv.ParseFloat(match, 32)
	if err != nil {
		return 0, false
	}
	return float32(parsed), true
}

// NextExtraNumber returns the number for a chapter without one that comes right after last,
// e.g. an "Extra" after chapter 5 is 5.5, after 5.5 it's 5.6.
func NextExtraNumber(last float32) float32 {
	return NextFreeExtraNumber(last, nil)
}

// NextFreeExtraNumber is NextExtraNumber skipping the numbers in taken: after a whole N it tries
// N.5, else last+0.1, then steps by 0.1, then .91, .92, ... (and .991 after .99). It stays below
// the next whole number a chapter has; when no chapter has N+1, a long run of extras goes on
// past it (N+1.5, N+1.6, ...), never on the whole number itself.
func NextFreeExtraNumber(last float32, taken map[float32]bool) float32 {
	// exact decimals: work on the number ×10^6, parsed from its shortest representation
	const scale = 1_000_000
	scaled := func(n float32) int64 {
		f, _ := strconv.ParseFloat(FormattedFloat(n), 64)
		return int64(math.Round(f * scale))
	}
	// past a whole number only when no chapter has it
	for base := scaled(last); ; base = (base/scale + 1) * scale {
		next := (base/scale + 1) * scale
		step := int64(scale / 10)
		candidate := base + step
		if base%scale == 0 {
			candidate = base + scale/2
		}
		for step > 0 {
			if candidate >= next {
				candidate -= step
				step /= 10
				candidate += step
				continue
			}
			n := float32(float64(candidate) / scale)
			// float32 can't hold every millionth of big numbers (it'd round to a taken or whole one)
			if scaled(n) == candidate && !taken[n] {
				return n
			}
			candidate += step
		}
		if taken[float32(next/scale)] {
			break
		}
	}
	// shortcut: about 40 extras in a row fit before a whole number a chapter has (float32),
	// later ones share the previous extra's number instead of taking a real chapter's; backlog
	// "extra chapters"
	return last
}

// EscapePath escapes each "/"-separated part of p, e.g. an id given by the user that goes in a
// url's path.
func EscapePath(p string) string {
	parts := strings.Split(p, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

// DateOf converts t to a metadata.Date.
// ParseStatus maps the status the sites show (asurascans, flamecomics, the scrapers) to the
// metadata one, "UNKNOWN" if it isn't one of them.
func ParseStatus(status string) metadata.Status {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "ongoing", "publishing", "releasing":
		return metadata.StatusReleasing
	case "completed", "complete", "finished":
		return metadata.StatusFinished
	case "hiatus", "on hold":
		return metadata.StatusHiatus
	case "dropped", "cancelled", "canceled":
		return metadata.StatusCancelled
	case "coming soon", "not yet released", "upcoming":
		return metadata.StatusNotYetReleased
	}
	return metadata.Status("UNKNOWN")
}

// CountryOfType: The country of origin (ISO 3166-1 alpha-2) of a manga, manhwa or manhua, ""
// for anything else.
func CountryOfType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "manga":
		return "JP"
	case "manhwa":
		return "KR"
	case "manhua":
		return "CN"
	}
	return ""
}

func DateOf(t time.Time) metadata.Date {
	return metadata.Date{Year: t.Year(), Month: int(t.Month()), Day: t.Day()}
}

// ImageExtension returns the extension of the image url (e.g. ".png"), ignoring its query.
func ImageExtension(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	ext := path.Ext(u.Path)
	if !ImageExtensionRegex.MatchString(ext) {
		return "", fmt.Errorf("invalid image extension %q in %q", ext, rawURL)
	}
	return ext, nil
}
