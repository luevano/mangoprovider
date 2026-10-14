package mangoprovider

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/luevano/libmangal/metadata"
)

var (
	relativeDateRegex = regexp.MustCompile(`(?i)\b(\d+|an?|one)\s*(seconds?|secs?|s|minutes?|mins?|m|hours?|hrs?|h|days?|d|weeks?|wks?|w|months?|mos?|years?|yrs?|y)\.?\s*ago\b`)
	ordinalRegex      = regexp.MustCompile(`(?i)\b(\d{1,2})(st|nd|rd|th)\b`)
)

// ParseDate: The date in text, either relative to now as sites show recent ones ("just now",
// "today", "yesterday", "an hour ago", "2 days ago", "3d ago") or absolute in one of the
// layouts (time.Parse's, month names in any case, "1st" is "1"). Zero and false if it's
// neither.
func ParseDate(text string, now time.Time, layouts ...string) (metadata.Date, bool) {
	text = CleanString(text)
	if text == "" {
		return metadata.Date{}, false
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, text); err == nil {
			return DateOf(t), true
		}
	}
	if stripped := ordinalRegex.ReplaceAllString(text, "$1"); stripped != text {
		for _, layout := range layouts {
			if t, err := time.Parse(layout, stripped); err == nil {
				return DateOf(t), true
			}
		}
	}
	if t, ok := relativeDate(strings.ToLower(text), now); ok {
		return DateOf(t), true
	}
	return metadata.Date{}, false
}

func relativeDate(text string, now time.Time) (time.Time, bool) {
	switch {
	case strings.HasPrefix(text, "just now"), strings.HasPrefix(text, "today"):
		return now, true
	case strings.HasPrefix(text, "yesterday"):
		return now.AddDate(0, 0, -1), true
	}
	m := relativeDateRegex.FindStringSubmatch(text)
	if m == nil {
		return time.Time{}, false
	}
	n := 1
	if number, err := strconv.Atoi(m[1]); err == nil {
		n = number
	}
	switch unit := strings.TrimSuffix(m[2], "s"); {
	case unit == "" || unit == "sec" || unit == "second":
		return now.Add(-time.Duration(n) * time.Second), true
	case unit == "m" || unit == "min" || unit == "minute":
		return now.Add(-time.Duration(n) * time.Minute), true
	case unit == "h" || unit == "hr" || unit == "hour":
		return now.Add(-time.Duration(n) * time.Hour), true
	case unit == "d" || unit == "day":
		return now.AddDate(0, 0, -n), true
	case unit == "w" || unit == "wk" || unit == "week":
		return now.AddDate(0, 0, -7*n), true
	case unit == "mo" || unit == "month":
		return now.AddDate(0, -n, 0), true
	case unit == "y" || unit == "yr" || unit == "year":
		return now.AddDate(-n, 0, 0), true
	}
	return time.Time{}, false
}
