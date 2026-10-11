package mangabox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mango "github.com/luevano/mangoprovider"
	"golang.org/x/time/rate"
)

// trimmed copies of what the sites serve (checked 2026-10-11)
const (
	challenge  = `<!DOCTYPE html><html lang="en-US"><head><title>Just a moment...</title></head></html>`
	searchPage = `<div class="panel_story_list"><div class="story_item"><a href="https://site/manga/solo-leveling"><img src="https://img/thumb/solo-leveling.webp"></a>
<div class="story_item_right"><h3 class="story_name"><a href="https://site/manga/solo-leveling">Solo Leveling</a></h3></div></div></div>`
	chaptersJSON = `{"success":true,"data":{"chapters":[{"chapter_name":"Chapter 1","chapter_slug":"chapter-1","chapter_num":1,"updated_at":"2025-09-20T14:03:09.000000Z"},{"chapter_name":"Chapter 0.1","chapter_slug":"chapter-0-1","chapter_num":0.1,"updated_at":"2025-09-19T14:03:09.000000Z"}]}}`
	chapterPage  = `<script>var cdns = ["https:\/\/img-r1.2xstorage.com\/"]; var chapterImages = ["solo-leveling\/1\/0.webp","solo-leveling\/1\/1.webp"];</script>`
)

func newTest(t *testing.T, flaresolverr bool, pages map[string]string) *mangabox {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := pages[r.URL.Path]
		switch {
		case !ok:
			http.NotFound(w, r)
		case body == challenge:
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(body))
		default:
			w.Write([]byte(body))
		}
	}))
	t.Cleanup(server.Close)
	m := &mangabox{
		info:    MangabatInfo,
		client:  server.Client(),
		limiter: rate.NewLimiter(rate.Inf, 0),
		base:    server.URL + "/",
	}
	m.headless.UseFlaresolverr = flaresolverr
	return m
}

func TestSearch(t *testing.T) {
	m := newTest(t, false, map[string]string{"/search/story/solo_leveling": searchPage})
	mangas, err := m.search(context.Background(), "Solo Leveling!")
	if err != nil || len(mangas) != 1 {
		t.Fatalf("got %v, %v", mangas, err)
	}
	if manga := mangas[0].(*mango.Manga); manga.ID != "solo-leveling" || manga.Title != "Solo Leveling" {
		t.Errorf("got %+v", manga)
	}
}

func TestChallengeWithoutFlaresolverr(t *testing.T) {
	m := newTest(t, false, map[string]string{"/search/story/solo_leveling": challenge})
	_, err := m.search(context.Background(), "solo leveling")
	if err == nil || !strings.Contains(err.Error(), "use_flaresolverr") {
		t.Errorf("got %v, want a hint to enable flaresolverr", err)
	}
}

func TestChapters(t *testing.T) {
	for name, body := range map[string]string{
		"json": chaptersJSON,
		// what flaresolverr returns: the browser's view of the json
		"flaresolverr": `<html><head></head><body><pre>` + strings.ReplaceAll(chaptersJSON, `"`, "&quot;") + `</pre></body></html>`,
	} {
		m := newTest(t, false, map[string]string{"/api/manga/solo-leveling/chapters": body})
		chapters, err := m.chapters(context.Background(), mango.Volume{Manga_: &mango.Manga{ID: "solo-leveling"}})
		if err != nil || len(chapters) != 2 {
			t.Fatalf("%s: got %v, %v", name, chapters, err)
		}
		first := chapters[0].(*mango.Chapter)
		if first.Number != 0.1 || first.ID != "chapter-0-1" || first.Date.Year != 2025 || !strings.HasSuffix(first.URL, "/manga/solo-leveling/chapter-0-1") {
			t.Errorf("%s: first chapter %+v, want 0.1 (oldest first)", name, first)
		}
	}
}

func TestChapterPages(t *testing.T) {
	m := newTest(t, false, map[string]string{"/manga/solo-leveling/chapter-1": chapterPage})
	chapter := mango.Chapter{ID: "chapter-1", Volume_: &mango.Volume{Manga_: &mango.Manga{ID: "solo-leveling"}}}
	pages, err := m.ChapterPages(context.Background(), mango.Store{}, chapter)
	if err != nil || len(pages) != 2 {
		t.Fatalf("got %v, %v", pages, err)
	}
	if url := pages[1].(*mango.Page).URL; url != "https://img-r1.2xstorage.com/solo-leveling/1/1.webp" {
		t.Errorf("got %s", url)
	}
}

func TestSearchAlias(t *testing.T) {
	for query, want := range map[string]string{
		"Solo Leveling":             "solo_leveling",
		"Kaguya-sama: Love is War!": "kaguya_sama_love_is_war",
		"Đường về":                  "duong_ve",
	} {
		if got := searchAlias(query); got != want {
			t.Errorf("%q: got %q, want %q", query, got, want)
		}
	}
}
