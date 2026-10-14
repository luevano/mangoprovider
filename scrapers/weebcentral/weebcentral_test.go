package weebcentral

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/scrapertest"
	"golang.org/x/time/rate"
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

func TestScrape(t *testing.T) {
	const id = "01J76XY7E9FNDZ1DBBM6PBJPFK"
	// the site's own urls, built from Info.Website, would go to the live site
	var base string
	config := Config(nil)
	config.ChaptersURL = func(manga mango.Manga) string { return base + "series/" + manga.ID + "/full-chapter-list" }
	config.PagesURL = func(chapter mango.Chapter) string { return base + "chapters/" + chapter.ID + "/images" }
	config.RateLimit = rate.Inf
	s, base := scrapertest.New(t, config, map[string]string{
		"/search/data":                                "search.html",
		"/search/data?offset=32&limit=32":             "search-2.html",
		"/series/" + id:                               "series.html",
		"/series/" + id + "/One-Piece":                "series.html",
		"/series/" + id + "/full-chapter-list":        "chapters.html",
		"/chapters/01J76XZ2DN3PRYRCWZSV1WB0TB/images": "images.html",
	})
	ctx := context.Background()

	mangas, err := s.SearchMangas(ctx, mango.Store{}, "one piece")
	if err != nil {
		t.Fatal(err)
	}
	// the odd link is skipped, page 2 repeats one
	want := []struct{ title, id string }{
		{"One Piece", id},
		{"One Piece Party", "01J76XYCERXE60T7FKXVCCAQ0H"},
		{"One Piece: Ace", "01J76XYCFE7Y9HJ5BNTS9Z6RXA"},
	}
	if len(mangas) != len(want) {
		t.Fatalf("got %v, want %d mangas", mangas, len(want))
	}
	for i, w := range want {
		if info := mangas[i].Info(); info.Title != w.title || info.ID != w.id || info.Cover != "https://temp.test/cover/normal/"+w.id+".jpg" {
			t.Errorf("manga %d: got %+v, want %+v", i, info, w)
		}
	}

	manga, found, err := s.MangaByID(ctx, mango.Store{}, id)
	if err != nil || !found {
		t.Fatalf("by id: %v, %v, %v", manga, found, err)
	}
	if info := manga.Info(); info.Title != "One Piece" || info.ID != id {
		t.Errorf("by id: got %+v", info)
	}
	meta := manga.Metadata()
	if err := metadata.Validate(meta); err != nil {
		t.Errorf("by id: %s", err)
	}
	if meta.Title() != "One Piece" || strings.Join(meta.AlternateTitles(), ",") != "ONE PIECE,ワンピース" ||
		meta.StartDate().Year != 1997 || meta.Status() != metadata.StatusReleasing || strings.Join(meta.Authors(), ",") != "ODA Eiichiro" ||
		strings.Join(meta.Genres(), ",") != "Action,Shounen" || meta.Format() != "Manga" || meta.Country() != "JP" ||
		meta.Cover() != "https://temp.test/cover/normal/"+id+".webp" || meta.ID().Code != "wc" || meta.ID().Raw != id ||
		meta.Description() != "As a child, Monkey D. Luffy dreamed of becoming the King of the Pirates." {
		t.Errorf("by id: got %+v", meta)
	}
	if ids := fmt.Sprint(meta.ExtraIDs()); len(meta.ExtraIDs()) != 2 || meta.ExtraIDs()[0].Value() != 30013 || meta.ExtraIDs()[1].Value() != 13 {
		t.Errorf("by id: extra ids %s", ids)
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
	// oldest first, the odd link skipped, the prologue before chapter 1
	wantChapters := []struct {
		number   float32
		id, date string
	}{
		{0.01, "01J76XZ2DN3PRYRCWZSV1WB0TA", ""},
		{1, "01J76XZ2DN3PRYRCWZSV1WB0TB", "1997-07-22"},
		{2, "01J76XZ2DN3PRYRCWZSV1WB0TC", "1997-08-04"},
	}
	if len(chapters) != len(wantChapters) {
		t.Fatalf("got %v, want %d chapters", chapters, len(wantChapters))
	}
	for i, w := range wantChapters {
		c := chapters[i].(*mango.Chapter)
		if c.Number != w.number || c.ID != w.id || c.Date.String() != w.date || c.URL != base+"chapters/"+w.id {
			t.Errorf("chapter %d: got %+v %s, want %+v", i, c, c.Date, w)
		}
	}

	pages, err := s.ChapterPages(ctx, mango.Store{}, *chapters[1].(*mango.Chapter))
	if err != nil || len(pages) != 2 {
		t.Fatalf("pages: %v, %v", pages, err)
	}
	if p := pages[1].(*mango.Page); p.URL != "https://scans.test/manga/One-Piece/0001-002.png" || p.Ext != ".png" {
		t.Errorf("page 2: got %+v", p)
	}
}

func TestRateLimit(t *testing.T) {
	// keiyoushi's rateLimit(1, 2.seconds) on the site's host
	if c := Config(nil); c.RateLimit != rate.Every(2*time.Second) || c.RateBurst != 1 {
		t.Errorf("got %v/s burst %d, want 1 per 2s", c.RateLimit, c.RateBurst)
	}
}
