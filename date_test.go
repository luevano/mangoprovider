package mangoprovider

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	now := time.Date(2025, 10, 13, 12, 0, 0, 0, time.UTC)
	layouts := []string{"Jan 2, 06", "January 2, 2006", time.RFC3339}
	for _, tt := range []struct{ text, want string }{
		{"Oct 10, 25", "2025-10-10"},
		{"Apr 3, 25", "2025-04-03"},
		{"  MAY 31, 23 ", "2023-05-31"},
		{"March 3rd, 2024", "2024-03-03"},
		{"2024-03-05T10:00:00+00:00", "2024-03-05"},
		{"just now", "2025-10-13"},
		{"Today", "2025-10-13"},
		{"yesterday", "2025-10-12"},
		{"30 minutes ago", "2025-10-13"},
		{"13 hours ago", "2025-10-12"},
		{"an hour ago", "2025-10-13"},
		{"a day ago", "2025-10-12"},
		{"2 days ago", "2025-10-11"},
		{"3d ago", "2025-10-10"},
		{"Updated 1 week ago", "2025-10-06"},
		{"2 months ago", "2025-08-13"},
		{"one year ago", "2024-10-13"},
		{"soon", ""},
		{"", ""},
		{"2 days", ""},
	} {
		got, ok := ParseDate(tt.text, now, layouts...)
		if got.String() != tt.want || ok != (tt.want != "") {
			t.Errorf("%q: wanted %q, got %q (%v)", tt.text, tt.want, got, ok)
		}
	}
}
