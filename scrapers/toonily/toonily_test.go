package toonily

import (
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

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

func TestDate(t *testing.T) {
	now := time.Date(2025, 10, 13, 12, 0, 0, 0, time.UTC)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<ul>
<li><span class="chapter-release-date"><i>Oct 10, 25</i></span></li>
<li><span class="chapter-release-date"><a href="#" title="2 hours ago" class="c-new-tag"><img src="new.png"></a></span></li>
<li><span class="chapter-release-date"><a href="#" title="3 days ago" class="c-new-tag"></a></span></li>
<li><span class="chapter-release-date"><i>yesterday</i></span></li>
<li><span class="chapter-release-date"><i>soon</i></span></li>
<li><span class="chapter-release-date"><a title="30 minutes ago"></a></span></li>
<li></li>
</ul>`))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2025-10-10", "2025-10-13", "2025-10-10", "2025-10-12", "", "2025-10-13", ""}
	doc.Find("li").Each(func(i int, s *goquery.Selection) {
		if got := releaseDate(s, now).String(); got != want[i] {
			t.Errorf("chapter %d: wanted %q, got %q", i, want[i], got)
		}
	})
}
