package toonily

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/scrapertest"
)

func TestIDs(t *testing.T) {
	config := Config(nil)
	for _, tt := range []struct{ url, manga, chapter string }{
		{"https://toonily.com/serie/solo-leveling-574def58/", "solo-leveling-574def58", ""},
		{"https://toonily.com/serie/solo-leveling-574def58/chapter-1/", "solo-leveling-574def58", "chapter-1"},
		// the old path still gives the ids
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

func TestScrape(t *testing.T) {
	config := Config(nil)
	// 25 a page on the site, 3 in the fixtures
	config.SearchPagination.PageSize = 3
	s, base := scrapertest.New(t, config, map[string]string{
		"/wp-admin/admin-ajax.php?action=madara_load_more&page=0&vars[s]=solo leveling": "search.html",
		"/wp-admin/admin-ajax.php?action=madara_load_more&page=1&vars[s]=solo leveling": "search-2.html",
		"/serie/solo-leveling/":               "manga.html",
		"/serie/solo-leveling/ajax/chapters/": "chapters.html",
		"/serie/solo-leveling/chapter-1/":     "chapter.html",
	})
	ctx := context.Background()

	mangas, err := s.SearchMangas(ctx, mango.Store{}, "Solo: Leveling!")
	if err != nil {
		t.Fatal(err)
	}
	// the odd link is skipped, page 2 repeats one
	want := []struct{ title, id, cover string }{
		{"Solo Leveling", "solo-leveling", "https://cdn.test/toonily/solo-leveling-350x476.jpg"},
		{"Solo Leveling: Ragnarok", "solo-leveling-ragnarok", "https://cdn.test/toonily/ragnarok-224x320.jpg"},
		{"Solo Leveling: Side Story", "solo-leveling-side-story", "https://cdn.test/toonily/side-story-224x320.jpg"},
	}
	if len(mangas) != len(want) {
		t.Fatalf("got %v, want %d mangas", mangas, len(want))
	}
	for i, w := range want {
		if info := mangas[i].Info(); info.Title != w.title || info.ID != w.id || info.Cover != w.cover || info.URL != base+"serie/"+w.id+"/" {
			t.Errorf("manga %d: got %+v, want %+v", i, info, w)
		}
	}

	manga, found, err := s.MangaByID(ctx, mango.Store{}, "solo-leveling")
	if err != nil || !found {
		t.Fatalf("by id: %v, %v, %v", manga, found, err)
	}
	// without the "END" badge
	if info := manga.Info(); info.Title != "Solo Leveling" || info.Cover != "https://cdn.test/toonily/solo-leveling-224x320.jpg" {
		t.Errorf("by id: got %+v", info)
	}

	meta := manga.Metadata()
	if meta.Title() != "Solo Leveling" || strings.Join(meta.AlternateTitles(), ",") != "나 혼자만 레벨업" ||
		strings.Join(meta.Authors(), ",") != "Chu-Gong" || strings.Join(meta.Artists(), ",") != "GEE So-Lyung,REDICE studio" ||
		strings.Join(meta.Genres(), ",") != "Action,Fantasy" || meta.Status() != metadata.StatusFinished || meta.Publisher() != "Kakao" ||
		meta.Description() != "When inter-dimensional “gates” opened up, hunters were humanity’s only hope.\n\nSung Jinwoo is the weakest of them all." ||
		meta.Cover() != "https://cdn.test/toonily/solo-leveling-224x320.jpg" || meta.ID().Code != "tl" || meta.ID().Raw != "solo-leveling" {
		t.Errorf("by id: got %+v", meta)
	}
	// no year on the page, until the chapters are read
	if meta.StartDate().Year != 0 {
		t.Errorf("by id: start %v", meta.StartDate())
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
		if c.Number != w.number || c.ID != w.id || c.Title != w.title || c.Date.String() != w.date || c.URL != base+"serie/solo-leveling/"+w.id+"/" || c.ScanlationGroup != "Toonily" {
			t.Errorf("chapter %d: got %+v %s, want %+v", i, c, c.Date, w)
		}
	}

	// the search result has the page's metadata now, started with the oldest chapter
	meta = mangas[0].Metadata()
	if err := metadata.Validate(meta); err != nil || meta.StartDate().String() != "2018-03-04" {
		t.Errorf("search result: %v, %+v", err, meta)
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
