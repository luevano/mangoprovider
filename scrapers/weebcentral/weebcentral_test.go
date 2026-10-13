package weebcentral

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

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

func TestDate(t *testing.T) {
	date := Config(nil).ChapterExtractor.Date
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<a><time datetime="2024-03-05T10:00:00.000Z"></time></a><a>no date</a>`))
	if err != nil {
		t.Fatal(err)
	}
	links := doc.Find("a")
	if got := date(links.First()); got.String() != "2024-03-05" {
		t.Errorf("wanted 2024-03-05, got %v", got)
	}
	if got := date(links.Last()); got.Year != 0 {
		t.Errorf("wanted no date, got %v", got)
	}
}
