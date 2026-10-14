package mangabox

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/progress"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/fetch"
	"golang.org/x/time/rate"
)

// reports records the statuses reported through the returned context.
func reports() (context.Context, *[]string) {
	var got []string
	ctx := progress.WithReporter(context.Background(), func(s progress.Status) { got = append(got, s.String()) })
	return ctx, &got
}

// trimmed copies of what the sites serve (checked 2026-10-11)
const (
	challenge  = `<!DOCTYPE html><html lang="en-US"><head><title>Just a moment...</title></head></html>`
	searchPage = `<div class="panel_story_list"><div class="story_item"><a href="https://site/manga/solo-leveling"><img src="https://img/thumb/solo-leveling.webp"></a>
<div class="story_item_right"><h3 class="story_name"><a href="https://site/manga/solo-leveling">Solo Leveling</a></h3></div></div></div>`
	chaptersJSON = `{"success":true,"data":{"chapters":[{"chapter_name":"Chapter 1","chapter_slug":"chapter-1","chapter_num":1,"updated_at":"2025-09-20T14:03:09.000000Z"},{"chapter_name":"Chapter 0.1","chapter_slug":"chapter-0-1","chapter_num":0.1,"updated_at":"2025-09-19T14:03:09.000000Z"}]}}`
	chapterPage  = `<script>var cdns = ["https:\/\/img-r1.2xstorage.com\/"]; var chapterImages = ["solo-leveling\/1\/0.webp","solo-leveling\/1\/1.webp"];</script>`
)

func newServer(t *testing.T, pages map[string]string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := pages[r.URL.Path]
		switch {
		case !ok:
			http.NotFound(w, r)
		case body == challenge:
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(body))
		case body == "down":
			w.WriteHeader(http.StatusServiceUnavailable)
		case body == "limited":
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			w.Write([]byte(body))
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// challengedBrowser: What rod gets on these sites, the challenge again.
type challengedBrowser struct{}

func (challengedBrowser) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusForbidden, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(challenge)), Request: r}, nil
}

func (challengedBrowser) Close() error { return nil }

func newTest(t *testing.T, flaresolverr bool, pages map[string]string, mirrors ...string) *mangabox {
	t.Helper()
	return newTestWith(t, mango.HeadlessOptions{UseFlaresolverr: flaresolverr}, pages, mirrors...)
}

func newTestWith(t *testing.T, headless mango.HeadlessOptions, pages map[string]string, mirrors ...string) *mangabox {
	t.Helper()
	info := MangabatInfo
	info.Website, info.Mirrors = newServer(t, pages).URL+"/", mirrors
	log := mango.NewProviderLogger(info)
	client, err := fetch.NewClient(http.DefaultClient, headless, log, fetch.Config{
		Strategy:   fetch.HeadlessFallback,
		NewBrowser: func() fetch.Browser { return challengedBrowser{} },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return &mangabox{
		info:    info,
		log:     log,
		client:  client,
		mirrors: mango.NewMirrors(info, "", log),
		cdnDown: map[string]bool{},
	}
}

func TestSearch(t *testing.T) {
	m := newTest(t, false, map[string]string{"/search/story/solo_leveling": searchPage})
	mangas, err := m.search(context.Background(), mango.Store{}, "Solo Leveling!")
	if err != nil || len(mangas) != 1 {
		t.Fatalf("got %v, %v", mangas, err)
	}
	if manga := mangas[0].(*mango.Manga); manga.ID != "solo-leveling" || manga.Title != "Solo Leveling" {
		t.Errorf("got %+v", manga)
	}
}

func TestChallengeWithoutFlaresolverr(t *testing.T) {
	m := newTest(t, false, map[string]string{"/search/story/solo_leveling": challenge})
	_, err := m.search(context.Background(), mango.Store{}, "solo leveling")
	if err == nil || !strings.Contains(err.Error(), "use_flaresolverr") {
		t.Errorf("got %v, want a hint to enable flaresolverr", err)
	}
}

func TestChallengeSolved(t *testing.T) {
	fs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out, _ := json.Marshal(map[string]any{"status": "ok", "solution": map[string]any{"status": 200, "response": searchPage}})
		w.Write(out)
	}))
	defer fs.Close()
	m := newTestWith(t, mango.HeadlessOptions{UseFlaresolverr: true, FlaresolverrURL: fs.URL}, map[string]string{"/search/story/solo_leveling": challenge})
	ctx, got := reports()
	mangas, err := m.search(ctx, mango.Store{}, "solo leveling")
	if err != nil || len(mangas) != 1 {
		t.Fatalf("got %v, %v", mangas, err)
	}
	if len(*got) != 2 || !strings.Contains((*got)[0], "cloudflare challenge") || !strings.Contains((*got)[1], "fetched through FlareSolverr") {
		t.Errorf("got reports %q, want the challenge and the fallback", *got)
	}
}

func TestChapters(t *testing.T) {
	for name, body := range map[string]string{
		"json": chaptersJSON,
		// what flaresolverr returns: the browser's view of the json
		"flaresolverr": `<html><head></head><body><pre>` + strings.ReplaceAll(chaptersJSON, `"`, "&quot;") + `</pre></body></html>`,
	} {
		m := newTest(t, false, map[string]string{"/api/manga/solo-leveling/chapters": body})
		chapters, err := m.chapters(context.Background(), mango.Store{}, mango.Volume{Manga_: &mango.Manga{ID: "solo-leveling"}})
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

func TestMirrorFallback(t *testing.T) {
	mirror := newServer(t, map[string]string{"/search/story/solo_leveling": searchPage, "/manga/nope": "down"})
	m := newTest(t, false, map[string]string{"/search/story/solo_leveling": "down"}, mirror.URL+"/")
	ctx, got := reports()
	mangas, err := m.search(ctx, mango.Store{}, "solo leveling")
	if err != nil || len(mangas) != 1 {
		t.Fatalf("got %v, %v, want the mirror's results", mangas, err)
	}
	site, other := mango.Host(m.info.Website), mango.Host(mirror.URL)
	want := []string{
		"Mangabat: " + site + " down (503 service unavailable), trying " + other,
		"Mangabat: using " + other,
	}
	if !slices.Equal(*got, want) {
		t.Errorf("reported %q, want %q", *got, want)
	}

	// not found isn't the site being down: no fallback
	_, err = m.manga(context.Background(), mango.Store{}, "missing")
	if !errors.Is(err, mango.ErrNotFound) {
		t.Errorf("got %v, want not found from the first site", err)
	}
}

func TestChapterPagesFallbacks(t *testing.T) {
	m := newTest(t, false, map[string]string{
		"/manga/solo-leveling/chapter-1": `<script>var cdns = ["https:\/\/a.com\/","https:\/\/a.com"]; var backupImage = ["https:\/\/b.com\/"]; var chapterImages = ["solo-leveling\/1\/0.webp"];</script>`,
		"/manga/solo-leveling/chapter-2": `<script>var cdns = ["https:\/\/a.com\/"];</script><div class="container-chapter-reader"><img src="https://c.com/2/0.jpg"></div>`,
	})
	for id, want := range map[string][]string{
		"chapter-1": {"https://a.com/solo-leveling/1/0.webp", "https://b.com/solo-leveling/1/0.webp"},
		"chapter-2": {"https://c.com/2/0.jpg", "https://a.com/2/0.jpg"},
	} {
		chapter := mango.Chapter{ID: id, Volume_: &mango.Volume{Manga_: &mango.Manga{ID: "solo-leveling"}}}
		pages, err := m.ChapterPages(context.Background(), mango.Store{}, chapter)
		if err != nil || len(pages) != 1 {
			t.Fatalf("%s: got %v, %v", id, pages, err)
		}
		page := pages[0].(*mango.Page)
		if got := append([]string{page.URL}, page.Fallbacks...); !slices.Equal(got, want) {
			t.Errorf("%s: got %v, want %v", id, got, want)
		}
	}
}

func TestPageImageFallback(t *testing.T) {
	var brokenHits int
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		brokenHits++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(broken.Close)
	working := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Referer") != "https://site/" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Write([]byte("image " + r.URL.Path))
	}))
	t.Cleanup(working.Close)

	m := newTest(t, false, nil)
	ctx, got := reports()
	for _, path := range []string{"/1/0.webp", "/1/1.webp"} {
		page := mango.Page{
			URL:       broken.URL + path,
			Fallbacks: []string{working.URL + path},
			Headers:   map[string]string{"Referer": "https://site/"},
		}
		image, err := m.GetPageImage(ctx, http.DefaultClient, page)
		if err != nil || string(image) != "image "+path {
			t.Fatalf("%s: got %q, %v", path, image, err)
		}
	}
	// the second page went straight to the cdn that worked
	if brokenHits != 1 {
		t.Errorf("broken cdn hit %d times, want 1", brokenHits)
	}
	want := []string{"Mangabat: image cdn " + mango.Host(broken.URL) + " failed (500 internal server error), trying " + mango.Host(working.URL)}
	if !slices.Equal(*got, want) {
		t.Errorf("reported %q, want %q", *got, want)
	}

	_, err := m.GetPageImage(context.Background(), http.DefaultClient, mango.Page{URL: broken.URL + "/2/0.webp"})
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Errorf("got %v, want the cdn's error", err)
	}
}

func TestMangaByID(t *testing.T) {
	page := `<div class="manga-info-top"><ul class="manga-info-text"><li><h1>Solo Leveling</h1></li><li>Author(s) : Chugong</li><li>Status : Completed</li></ul></div>`
	m := newTest(t, false, map[string]string{"/manga/solo-leveling": page})
	manga, found, err := m.MangaByID(context.Background(), mango.Store{}, "solo-leveling")
	if err != nil || !found || manga.Info().Title != "Solo Leveling" || manga.Info().ID != "solo-leveling" {
		t.Fatalf("got %v, %v, %v", manga, found, err)
	}
	if manga, found, err := m.MangaByID(context.Background(), mango.Store{}, "nope"); err != nil || found {
		t.Errorf("unknown id: %v, %v, %v, want not found", manga, found, err)
	}
	if mangas, err := m.SearchMangas(context.Background(), mango.Store{}, "mid:solo-leveling"); err != nil || len(mangas) != 1 {
		t.Errorf("search by id: %v, %v", mangas, err)
	}
}

func TestGetRateLimited(t *testing.T) {
	m := newTest(t, false, map[string]string{"/manga/x": "limited"})
	_, _, err := m.get(context.Background(), mango.Store{}, "/manga/x")
	if !errors.Is(err, mango.ErrRateLimited) {
		t.Errorf("wanted a rate limit, got %v", err)
	}
}

func TestImagesThrottled(t *testing.T) {
	// keiyoushi limits Mangakakalot's images too (Mangakakalot.kt rateLimit(2, 1.seconds))
	for _, info := range []libmangal.ProviderInfo{MangabatInfo, MangakakalotInfo, ManganatoInfo} {
		if got, want := imageLimiter(info) != nil, info.ID == MangakakalotInfo.ID; got != want {
			t.Errorf("%s: image limiter %t, want %t", info.ID, got, want)
		}
	}
	image := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("png")) }))
	defer image.Close()
	m := newTest(t, false, nil)
	m.images = rate.NewLimiter(rate.Every(100*time.Millisecond), 1)
	start := time.Now()
	for range 3 {
		if _, err := m.GetPageImage(context.Background(), image.Client(), mango.Page{URL: image.URL + "/p.png"}); err != nil {
			t.Fatal(err)
		}
	}
	if took := time.Since(start); took < 200*time.Millisecond {
		t.Errorf("3 images took %v, want at least 200ms", took)
	}
}
