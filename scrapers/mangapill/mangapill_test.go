package mangapill

import (
	"context"
	"strings"
	"testing"

	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/scrapertest"
)

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

// mangapill has no dates, leaving them empty instead of the day it was scraped
func TestNoDate(t *testing.T) {
	if Config(nil).ChapterExtractor.Date != nil {
		t.Error("chapters shouldn't get a date")
	}
}

func TestScrape(t *testing.T) {
	s, base := scrapertest.New(t, Config(nil), map[string]string{
		"/search":            "search.html",
		"/search?page=2":     "search-2.html",
		"/manga/2/one-piece": "manga.html",
		"/chapters/2-10001000/one-piece-chapter-1": "chapter.html",
	})
	ctx := context.Background()

	mangas, err := s.SearchMangas(ctx, mango.Store{}, "one piece")
	if err != nil {
		t.Fatal(err)
	}
	// the odd link is skipped, page 2 repeats one
	want := []struct{ title, id, cover string }{
		{"One Piece", "2/one-piece", "https://cdn.test/mangapill/i/2.jpeg"},
		{"One Piece Party", "3260/one-piece-party", "https://cdn.test/mangapill/i/3260.jpeg"},
		{"One Piece: Ace", "7/one-piece-ace", "https://cdn.test/mangapill/i/7.jpeg"},
	}
	if len(mangas) != len(want) {
		t.Fatalf("got %v, want %d mangas", mangas, len(want))
	}
	for i, w := range want {
		if info := mangas[i].Info(); info.Title != w.title || info.ID != w.id || info.Cover != w.cover || info.URL != base+"manga/"+w.id {
			t.Errorf("manga %d: got %+v, want %+v", i, info, w)
		}
	}

	manga, found, err := s.MangaByID(ctx, mango.Store{}, "2/one-piece")
	if err != nil || !found {
		t.Fatalf("by id: %v, %v, %v", manga, found, err)
	}
	if info := manga.Info(); info.Title != "One Piece" || info.ID != "2/one-piece" || info.Cover != "https://cdn.test/mangapill/i/2.jpeg" {
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
	// oldest first, the odd link skipped
	wantChapters := []struct {
		number float32
		id     string
	}{{1, "2-10001000/one-piece-chapter-1"}, {1.5, "2-10001500/one-piece-chapter-1.5"}, {2, "2-10002000/one-piece-chapter-2"}}
	if len(chapters) != len(wantChapters) {
		t.Fatalf("got %v, want %d chapters", chapters, len(wantChapters))
	}
	for i, w := range wantChapters {
		c := chapters[i].(*mango.Chapter)
		if c.Number != w.number || c.ID != w.id || c.URL != base+"chapters/"+w.id || c.Date.Year != 0 || c.ScanlationGroup != "Mangapill" {
			t.Errorf("chapter %d: got %+v, want %+v", i, c, w)
		}
	}

	pages, err := s.ChapterPages(ctx, mango.Store{}, *chapters[0].(*mango.Chapter))
	if err != nil || len(pages) != 2 {
		t.Fatalf("pages: %v, %v", pages, err)
	}
	if p := pages[1].(*mango.Page); p.URL != "https://cdn.test/mangapill/file/2/10001000/2.jpeg" || p.Ext != ".jpeg" || p.Headers["Referer"] != chapters[0].Info().URL {
		t.Errorf("page 2: got %+v", p)
	}
}

func TestMetadata(t *testing.T) {
	s, base := scrapertest.New(t, Config(nil), map[string]string{
		"/search":            "search.html",
		"/search?page=2":     "search-2.html",
		"/manga/2/one-piece": "manga.html",
	})
	ctx := context.Background()
	store := mango.NewMemoryStore()

	manga, found, err := s.MangaByID(ctx, store, "2/one-piece")
	if err != nil || !found {
		t.Fatalf("by id: %v, %v, %v", manga, found, err)
	}
	meta := manga.Metadata()
	if meta.Title() != "One Piece" || meta.StartDate().Year != 1997 || meta.Status() != metadata.StatusReleasing ||
		strings.Join(meta.Genres(), ",") != "Action,Super Power" || meta.Format() != "manga" || meta.Country() != "JP" ||
		meta.Cover() != "https://cdn.test/mangapill/i/2.jpeg" || meta.URL() != base+"manga/2/one-piece" ||
		meta.ID().Raw != "2/one-piece" || meta.ID().Code != "mp" ||
		meta.Description() != "Gol D. Roger, the \"Pirate King,\" is executed.\n\n[Written by MAL Rewrite]" {
		t.Errorf("got %+v", meta)
	}
	// the site has no authors
	if err := metadata.Validate(meta); err == nil || !strings.Contains(err.Error(), "no authors") {
		t.Errorf("validate: %v", err)
	}

	// a search result gets it from the cache
	mangas, err := s.SearchMangas(ctx, store, "one piece")
	if err != nil || len(mangas) != 3 {
		t.Fatalf("search: %v, %v", mangas, err)
	}
	if got := mangas[0].Metadata().StartDate().Year; got != 1997 {
		t.Errorf("search result: start %d", got)
	}
	if got := mangas[1].Metadata().Title(); got != "" {
		t.Errorf("search result without a cached page: %q", got)
	}
}
