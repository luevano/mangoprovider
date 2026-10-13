package toonily

import "testing"

func TestIDs(t *testing.T) {
	config := Config(nil)
	for _, tt := range []struct{ url, manga, chapter string }{
		{"https://toonily.com/webtoon/solo-leveling/", "solo-leveling", ""},
		{"https://toonily.com/webtoon/solo-leveling/chapter-1/", "solo-leveling", "chapter-1"},
		// odd urls are skipped, not a panic
		{"https://toonily.com/", "", ""},
		{"/", "", ""},
	} {
		if got := config.MangaExtractor.ID(tt.url); got != tt.manga {
			t.Errorf("manga id of %q: wanted %q, got %q", tt.url, tt.manga, got)
		}
		if got := config.ChapterExtractor.ID(tt.url); got != tt.chapter {
			t.Errorf("chapter id of %q: wanted %q, got %q", tt.url, tt.chapter, got)
		}
	}
}
