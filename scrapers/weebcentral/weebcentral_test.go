package weebcentral

import "testing"

func TestIDs(t *testing.T) {
	config := Config(nil)
	for _, tt := range []struct{ url, manga, chapter string }{
		{"https://weebcentral.com/series/01J76XY7EF75DJNQCV04HTPDZK/Berserk", "01J76XY7EF75DJNQCV04HTPDZK", ""},
		{"https://weebcentral.com/chapters/01KX4BFP5E3FEZ50TRS752CT9V", "", "01KX4BFP5E3FEZ50TRS752CT9V"},
		// odd urls are skipped, not a panic
		{"https://weebcentral.com/series/", "", ""},
		{"https://weebcentral.com/", "", ""},
	} {
		if got := config.MangaExtractor.ID(tt.url); got != tt.manga {
			t.Errorf("manga id of %q: wanted %q, got %q", tt.url, tt.manga, got)
		}
		if got := config.ChapterExtractor.ID(tt.url); got != tt.chapter {
			t.Errorf("chapter id of %q: wanted %q, got %q", tt.url, tt.chapter, got)
		}
	}
}
