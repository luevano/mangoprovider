package mangapill

import "testing"

func TestIDs(t *testing.T) {
	config := Config(nil)
	for _, tt := range []struct{ url, manga, chapter string }{
		{"https://mangapill.com/manga/2/one-piece", "2/one-piece", "2/one-piece"},
		{"https://mangapill.com/chapters/2-10001000/one-piece-chapter-1", "2-10001000/one-piece-chapter-1", "2-10001000/one-piece-chapter-1"},
		// odd urls are skipped, not a panic
		{"https://mangapill.com/", "", ""},
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
