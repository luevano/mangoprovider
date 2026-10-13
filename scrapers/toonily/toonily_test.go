package toonily

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/scrapertest"
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

func TestScrape(t *testing.T) {
	s, base := scrapertest.New(t, Config(nil), map[string]string{
		"/search/solo-leveling":             "search.html",
		"/webtoon/solo-leveling":            "manga.html",
		"/webtoon/solo-leveling/":           "manga.html",
		"/webtoon/solo-leveling/chapter-1/": "chapter.html",
	})
	ctx := context.Background()

	mangas, err := s.SearchMangas(ctx, mango.Store{}, "solo leveling")
	if err != nil {
		t.Fatal(err)
	}
	// the odd link is skipped
	want := []struct{ title, id, cover string }{
		{"Solo Leveling", "solo-leveling", "https://cdn.test/toonily/solo-leveling-224x320.jpg"},
		{"Solo Leveling: Ragnarok", "solo-leveling-ragnarok", "https://cdn.test/toonily/ragnarok-224x320.jpg"},
	}
	if len(mangas) != len(want) {
		t.Fatalf("got %v, want %d mangas", mangas, len(want))
	}
	for i, w := range want {
		if info := mangas[i].Info(); info.Title != w.title || info.ID != w.id || info.Cover != w.cover || info.URL != base+"webtoon/"+w.id+"/" {
			t.Errorf("manga %d: got %+v, want %+v", i, info, w)
		}
	}

	manga, found, err := s.MangaByID(ctx, mango.Store{}, "solo-leveling")
	if err != nil || !found {
		t.Fatalf("by id: %v, %v, %v", manga, found, err)
	}
	// without the "HOT" badge
	if info := manga.Info(); info.Title != "Solo Leveling" || info.Cover != "https://cdn.test/toonily/solo-leveling-193x278.jpg" {
		t.Errorf("by id: got %+v", info)
	}

	m := *mangas[0].(*mango.Manga)
	volumes, err := s.MangaVolumes(ctx, mango.Store{}, m)
	if err != nil || len(volumes) != 1 {
		t.Fatalf("volumes: %v, %v", volumes, err)
	}
	chapters, err := s.VolumeChapters(ctx, mango.Store{}, *volumes[0].(*mango.Volume))
	if err != nil {
		t.Fatal(err)
	}
	// oldest first, the odd link skipped, "Side Story" right after 200
	newDate := mango.DateOf(time.Now().AddDate(0, 0, -2)).String()
	wantChapters := []struct {
		number    float32
		id, title string
		date      string
	}{
		{1, "chapter-1", "Chapter 1", "2018-03-04"},
		{200, "chapter-200", "Chapter 200", "2025-10-10"},
		{200.5, "side-story", "Side Story", ""},
		{201, "chapter-201", "Epilogue", newDate},
	}
	if len(chapters) != len(wantChapters) {
		t.Fatalf("got %v, want %d chapters", chapters, len(wantChapters))
	}
	for i, w := range wantChapters {
		c := chapters[i].(*mango.Chapter)
		if c.Number != w.number || c.ID != w.id || c.Title != w.title || c.Date.String() != w.date || c.URL != base+"webtoon/solo-leveling/"+w.id+"/" {
			t.Errorf("chapter %d: got %+v %s, want %+v", i, c, c.Date, w)
		}
	}

	pages, err := s.ChapterPages(ctx, mango.Store{}, *chapters[0].(*mango.Chapter))
	if err != nil || len(pages) != 2 {
		t.Fatalf("pages: %v, %v", pages, err)
	}
	for i, p := range pages {
		if want := fmt.Sprintf("https://cdn.test/toonily/chapters/solo-leveling/chapter-1/%02d.jpg", i+1); p.(*mango.Page).URL != want {
			t.Errorf("page %d: got %s, want %s", i, p.(*mango.Page).URL, want)
		}
	}
}
