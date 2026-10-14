package scraper

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/luevano/libmangal/mangadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/fetch"
)

func TestChapterPagesCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chapter" {
			// hang until the client gives up
			select {
			case <-r.Context().Done():
			case <-time.After(10 * time.Second):
			}
		}
	}))
	defer server.Close()

	config := &Configuration{
		Name:          "test",
		BaseURL:       server.URL,
		Parallelism:   1,
		PageExtractor: &PageExtractor{Selector: "img"},
	}
	s, err := NewScraper(context.Background(), config, mango.DefaultOptions(), nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = s.ChapterPages(ctx, mango.Store{}, mango.Chapter{URL: server.URL + "/chapter"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("wanted the context error, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("should stop right after the cancel, took %s", elapsed)
	}
}

// newTestScraper: Scraper for a local server answering /chapter with the given status and html.
func newTestScraper(t *testing.T, status int, html string, extractor *PageExtractor) (*Scraper, string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chapter" {
			w.WriteHeader(status)
			w.Write([]byte(html))
		}
	}))
	t.Cleanup(server.Close)

	config := &Configuration{Name: "test", BaseURL: server.URL, Parallelism: 1, PageExtractor: extractor}
	s, err := NewScraper(context.Background(), config, mango.DefaultOptions(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return s, server.URL + "/chapter"
}

func TestChapterPagesHTTPError(t *testing.T) {
	s, url := newTestScraper(t, http.StatusServiceUnavailable, "", &PageExtractor{Selector: "img"})

	_, err := s.ChapterPages(context.Background(), mango.Store{}, mango.Chapter{URL: url})
	if !errors.Is(err, mango.ErrSiteDown) || !strings.Contains(err.Error(), "GET "+url+": ") || !strings.Contains(err.Error(), "503 service unavailable") {
		t.Errorf("wanted the 503 error, got %v", err)
	}
}

func TestChapterPagesPanic(t *testing.T) {
	extractor := &PageExtractor{
		Selector: "img",
		URL: func(*goquery.Selection) string {
			var parts []string
			return parts[4] // like a site config indexing a url that changed
		},
	}
	s, url := newTestScraper(t, http.StatusOK, `<html><body><img src="1.png"></body></html>`, extractor)

	_, err := s.ChapterPages(context.Background(), mango.Store{}, mango.Chapter{URL: url})
	if err == nil || !strings.Contains(err.Error(), "parse "+url+": runtime error: index out of range") {
		t.Errorf("wanted the panic as an error, got %v", err)
	}
}

func TestChapterPagesEmpty(t *testing.T) {
	s, url := newTestScraper(t, http.StatusOK, `<html><body>no images</body></html>`, &PageExtractor{Selector: "img"})

	_, err := s.ChapterPages(context.Background(), mango.Store{}, mango.Chapter{URL: url})
	if !errors.Is(err, mango.ErrNotFound) {
		t.Errorf("wanted ErrNotFound, got %v", err)
	}
}

func TestNumberExtraChapters(t *testing.T) {
	numbers := []float32{-1, 1, 2, -1, -1, 3}
	var chapters []mangadata.Chapter
	for _, n := range numbers {
		chapters = append(chapters, &mango.Chapter{Number: n})
	}

	numberExtraChapters(chapters)

	want := []float32{0.5, 1, 2, 2.5, 2.6, 3}
	for i, chapter := range chapters {
		if got := chapter.Info().Number; got != want[i] {
			t.Errorf("chapter %d: wanted %v, got %v", i, want[i], got)
		}
	}
}

func TestMangaByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manga/1/one-piece":
			w.Write([]byte(`<html><main><h1> One Piece </h1><img src="cover.jpg"></main></html>`))
		case "/manga/2/changed":
			w.Write([]byte(`<html><div>new layout</div></html>`))
		case "/manga/4/gone":
			http.Redirect(w, r, "/404", http.StatusFound)
		case "/404":
			w.Write([]byte(`<html><main><h1>Page not found</h1></main></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	config := &Configuration{
		Name:        "test",
		BaseURL:     server.URL + "/",
		Parallelism: 1,
		GenerateSearchByIDURL: func(baseUrl string, id string) (string, error) {
			return baseUrl + "manga/" + id, nil
		},
		MangaByIDExtractor: &MangaByIDExtractor{
			Selector: "main",
			Title:    func(s *goquery.Selection) string { return s.Find("h1").Text() },
			Cover:    func(s *goquery.Selection) string { return s.Find("img").AttrOr("src", "") },
		},
	}
	s, err := NewScraper(context.Background(), config, mango.DefaultOptions(), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	manga, found, err := s.MangaByID(ctx, mango.Store{}, "/1/one-piece/")
	if err != nil || !found {
		t.Fatalf("got %v, %v, %v", manga, found, err)
	}
	if info := manga.Info(); info.Title != "One Piece" || info.ID != "1/one-piece" || info.Cover != "cover.jpg" {
		t.Errorf("got %+v", info)
	}
	// a query or fragment in the id stays in the path
	for _, id := range []string{"3/nope", "4/gone", "1/one-piece?page=2", "1/one-piece#x"} {
		if manga, found, err := s.MangaByID(ctx, mango.Store{}, id); err != nil || found {
			t.Errorf("%s: %v, %v, %v, want not found", id, manga, found, err)
		}
	}
	if _, found, err := s.MangaByID(ctx, mango.Store{}, "2/changed"); err == nil || found {
		t.Errorf("page without the manga: %v, %v, want an error", found, err)
	}
	if mangas, err := s.SearchMangas(ctx, mango.Store{}, "mid: 1/one-piece"); err != nil || len(mangas) != 1 {
		t.Errorf("search by id: %v, %v", mangas, err)
	}
}

// hostRecorder: Transport that records the Host header it gets, like the headless ones forward it.
func TestRequestHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	hosts := make(chan string, 1)
	record := func(next http.RoundTripper) http.RoundTripper {
		return fetch.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			hosts <- r.Header.Get("Host")
			return next.RoundTrip(r)
		})
	}
	config := &Configuration{Name: "test", BaseURL: server.URL, Parallelism: 1, PageExtractor: &PageExtractor{Selector: "img"}, Middlewares: []fetch.Middleware{record}}
	s, err := NewScraper(context.Background(), config, mango.DefaultOptions(), nil)
	if err != nil {
		t.Fatal(err)
	}
	s.ChapterPages(context.Background(), mango.Store{}, mango.Chapter{URL: server.URL + "/chapter"})

	select {
	case host := <-hosts:
		if host != "" {
			t.Errorf("the Host header should be left to net/http, got %q", host)
		}
	default:
		t.Error("the chapter wasn't requested")
	}
}

func TestChapterPagesAbsoluteURL(t *testing.T) {
	extractor := &PageExtractor{
		Selector: "img",
		URL:      func(s *goquery.Selection) string { return s.AttrOr("src", "") },
	}
	s, url := newTestScraper(t, http.StatusOK, `<html><body><img src="/images/1.png"><img src="https://cdn.test/2.png"></body></html>`, extractor)

	pages, err := s.ChapterPages(context.Background(), mango.Store{}, mango.Chapter{URL: url})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{strings.TrimSuffix(url, "/chapter") + "/images/1.png", "https://cdn.test/2.png"}
	for i, page := range pages {
		if got := page.(*mango.Page).URL; got != want[i] {
			t.Errorf("page %d: wanted %q, got %q", i, want[i], got)
		}
	}
}

func TestThrottle(t *testing.T) {
	s, url := newTestScraper(t, http.StatusOK, `<html><body><img src="1.png"></body></html>`, &PageExtractor{
		Selector: "img",
		URL:      func(s *goquery.Selection) string { return s.AttrOr("src", "") },
	})
	start := time.Now()
	for range requestBurst + 2 {
		if _, err := s.ChapterPages(context.Background(), mango.Store{}, mango.Chapter{URL: url}); err != nil {
			t.Fatal(err)
		}
	}
	if took, want := time.Since(start), 2*requestInterval; took < want {
		t.Errorf("%d requests took %v, want at least %v", requestBurst+2, took, want)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.ChapterPages(ctx, mango.Store{}, mango.Chapter{URL: url}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled while throttled: wanted the context error, got %v", err)
	}
}

func TestParallelism(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()
	for _, tt := range []struct{ option, want uint8 }{{15, maxParallelism}, {2, 2}} {
		options := mango.DefaultOptions()
		options.Parallelism = tt.option
		config := &Configuration{Name: "test", BaseURL: server.URL}
		if _, err := NewScraper(context.Background(), config, options, nil); err != nil {
			t.Fatal(err)
		}
		if config.Parallelism != tt.want {
			t.Errorf("option %d: wanted %d, got %d", tt.option, tt.want, config.Parallelism)
		}
	}
}

func TestSkipNoID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body><a href="/manga/1">One</a><a href="/odd">Odd</a><a href="/manga/2">Two</a></body></html>`))
	}))
	defer server.Close()

	id := func(u string) string {
		if _, id, ok := strings.Cut(u, "/manga/"); ok {
			return id
		}
		return ""
	}
	config := &Configuration{
		Name:        "test",
		BaseURL:     server.URL + "/",
		Parallelism: 1,
		GenerateSearchURL: func(baseUrl, query string) (string, error) {
			return baseUrl + "search", nil
		},
		MangaExtractor: &MangaExtractor{
			Selector: "a",
			Title:    func(s *goquery.Selection) string { return s.Text() },
			URL:      func(s *goquery.Selection) string { return s.AttrOr("href", "") },
			ID:       id,
			Cover:    func(*goquery.Selection) string { return "" },
		},
		ChapterExtractor: &ChapterExtractor{
			Selector: "a",
			Title:    func(s *goquery.Selection) string { return s.Text() },
			URL:      func(s *goquery.Selection) string { return s.AttrOr("href", "") },
			ID:       id,
		},
	}
	s, err := NewScraper(context.Background(), config, mango.DefaultOptions(), nil)
	if err != nil {
		t.Fatal(err)
	}

	mangas, err := s.SearchMangas(context.Background(), mango.Store{}, "x")
	if err != nil || len(mangas) != 2 || mangas[1].Info().ID != "2" {
		t.Errorf("wanted the odd manga skipped, got %v, %v", mangas, err)
	}
	volume := mango.Volume{Number: 1, Manga_: &mango.Manga{URL: server.URL + "/manga/1"}}
	chapters, err := s.VolumeChapters(context.Background(), mango.Store{}, volume)
	if err != nil || len(chapters) != 2 {
		t.Errorf("wanted the odd chapter skipped, got %v, %v", chapters, err)
	}
}

func TestRequestHeaders(t *testing.T) {
	got := make(chan http.Header, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			got <- r.Header.Clone()
		}
		w.Write([]byte(`<html><body><a href="/manga/1">One</a><img src="1.png"></body></html>`))
	}))
	defer server.Close()

	config := &Configuration{
		Name:              "test",
		BaseURL:           server.URL + "/",
		Parallelism:       1,
		Headers:           map[string]string{"Cookie": "toonily-mature=1"},
		GenerateSearchURL: func(baseUrl, query string) (string, error) { return baseUrl + "search", nil },
		MangaExtractor: &MangaExtractor{
			Selector: "a",
			Title:    func(s *goquery.Selection) string { return s.Text() },
			URL:      func(s *goquery.Selection) string { return s.AttrOr("href", "") },
			ID:       func(u string) string { return u },
			Cover:    func(*goquery.Selection) string { return "" },
		},
		PageExtractor: &PageExtractor{Selector: "img", URL: func(s *goquery.Selection) string { return s.AttrOr("src", "") }},
	}
	options := mango.DefaultOptions()
	options.UserAgent = "test-agent"
	s, err := NewScraper(context.Background(), config, options, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SearchMangas(context.Background(), mango.Store{}, "x"); err != nil {
		t.Fatal(err)
	}
	chapterURL := server.URL + "/chapter/1"
	if _, err := s.ChapterPages(context.Background(), mango.Store{}, mango.Chapter{URL: chapterURL}); err != nil {
		t.Fatal(err)
	}

	for _, referer := range []string{config.BaseURL, chapterURL} {
		h := <-got
		if h.Get("Referer") != referer || h.Get("User-Agent") != "test-agent" || h.Get("Cookie") != "toonily-mature=1" || h.Get("Accept") != "text/html" {
			t.Errorf("wanted referer %s, got headers %v", referer, h)
		}
	}
}

func TestRedirectedPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manga/old":
			http.Redirect(w, r, "/new/manga/1/", http.StatusMovedPermanently)
		case "/new/manga/1/":
			w.Write([]byte(`<html><body><a href="chapter-1">Chapter 1</a><a href="#top">Top</a></body></html>`))
		case "/based":
			w.Write([]byte(`<html><head><base href="/cdn/"></head><body><img src="1.png"></body></html>`))
		}
	}))
	defer server.Close()

	config := &Configuration{
		Name:        "test",
		BaseURL:     server.URL + "/",
		Parallelism: 1,
		ChapterExtractor: &ChapterExtractor{
			Selector: "a",
			Title:    func(s *goquery.Selection) string { return s.Text() },
			URL:      func(s *goquery.Selection) string { return s.AttrOr("href", "") },
			ID:       func(u string) string { return u },
		},
		PageExtractor: &PageExtractor{Selector: "img", URL: func(s *goquery.Selection) string { return s.AttrOr("src", "") }},
	}
	s, err := NewScraper(context.Background(), config, mango.DefaultOptions(), nil)
	if err != nil {
		t.Fatal(err)
	}

	volume := mango.Volume{Number: 1, Manga_: &mango.Manga{URL: server.URL + "/manga/old"}}
	chapters, err := s.VolumeChapters(context.Background(), mango.Store{}, volume)
	// relative to where the redirect ended, the fragment-only link has no url
	if err != nil || len(chapters) != 1 || chapters[0].Info().URL != server.URL+"/new/manga/1/chapter-1" {
		t.Errorf("got %v, %v", chapters, err)
	}

	pages, err := s.ChapterPages(context.Background(), mango.Store{}, mango.Chapter{URL: server.URL + "/based"})
	if err != nil || len(pages) != 1 || pages[0].(*mango.Page).URL != server.URL+"/cdn/1.png" {
		t.Errorf("relative to <base href>: got %v, %v", pages, err)
	}
}

func TestCharset(t *testing.T) {
	// "Capítulo" in latin-1
	latin1 := "Cap\xedtulo 1"
	for name, tt := range map[string]struct{ contentType, html string }{
		"utf-8":           {"text/html; charset=utf-8", "<p>Capítulo 1</p>"},
		"undeclared":      {"text/html", "<p>Capítulo 1</p>"},
		"content-type":    {"text/html; charset=iso-8859-1", "<p>" + latin1 + "</p>"},
		"meta charset":    {"text/html", `<head><meta charset="windows-1252"></head><p>` + latin1 + "</p>"},
		"meta http-equiv": {"text/html", `<head><meta http-equiv="Content-Type" content="text/html; charset=ISO-8859-1"></head><p>` + latin1 + "</p>"},
	} {
		doc, err := parseHTML([]byte(tt.html), tt.contentType)
		if err != nil {
			t.Fatal(err)
		}
		if got := doc.Find("p").Text(); got != "Capítulo 1" {
			t.Errorf("%s: got %q", name, got)
		}
	}
}

func TestCookies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manga/old":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc", Path: "/"})
			http.Redirect(w, r, "/manga/new", http.StatusFound)
			return
		case "/":
			return
		}
		if c, err := r.Cookie("session"); err != nil || c.Value != "abc" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Write([]byte(`<html><body><a href="chapter-1">Chapter 1</a><img src="1.png"></body></html>`))
	}))
	defer server.Close()

	newConfig := func() *Configuration {
		return &Configuration{
			Name:        "test",
			BaseURL:     server.URL + "/",
			Parallelism: 1,
			ChapterExtractor: &ChapterExtractor{
				Selector: "a",
				Title:    func(s *goquery.Selection) string { return s.Text() },
				URL:      func(s *goquery.Selection) string { return s.AttrOr("href", "") },
				ID:       func(u string) string { return u },
			},
			PageExtractor: &PageExtractor{Selector: "img", URL: func(s *goquery.Selection) string { return s.AttrOr("src", "") }},
		}
	}
	options := mango.DefaultOptions()
	s, err := NewScraper(context.Background(), newConfig(), options, nil)
	if err != nil {
		t.Fatal(err)
	}
	if options.HTTPClient.Jar != nil {
		t.Error("the shared client got a cookie jar")
	}

	// the cookie set on the redirect reaches its target and the next requests
	volume := mango.Volume{Number: 1, Manga_: &mango.Manga{URL: server.URL + "/manga/old"}}
	if chapters, err := s.VolumeChapters(context.Background(), mango.Store{}, volume); err != nil || len(chapters) != 1 {
		t.Fatalf("got %v, %v", chapters, err)
	}
	if _, err := s.ChapterPages(context.Background(), mango.Store{}, mango.Chapter{URL: server.URL + "/chapter/1"}); err != nil {
		t.Errorf("next request without the cookie: %v", err)
	}

	// one jar per scraper
	other, err := NewScraper(context.Background(), newConfig(), options, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.ChapterPages(context.Background(), mango.Store{}, mango.Chapter{URL: server.URL + "/chapter/1"}); err == nil {
		t.Error("another scraper got the cookie")
	}
}

func TestFetchCloudflareFallback(t *testing.T) {
	var destroyed bool
	fs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "sessions.destroy") {
			destroyed = true
		}
		w.Write([]byte(`{"status": "ok", "solution": {"status": 200, "response": "<html><body><img src=\"/1.png\"></body></html>"}}`))
	}))
	defer fs.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chapter" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`<html><head><title>Just a moment...</title></head></html>`))
		}
	}))
	defer server.Close()

	options := mango.DefaultOptions()
	options.Headless = mango.HeadlessOptions{UseFlaresolverr: true, FlaresolverrURL: fs.URL}
	config := &Configuration{
		Name: "test", BaseURL: server.URL, Fetch: fetch.CloudflareFallback,
		PageExtractor: &PageExtractor{Selector: "img", URL: func(s *goquery.Selection) string { return s.AttrOr("src", "") }},
	}
	s, err := NewScraper(context.Background(), config, options, nil)
	if err != nil {
		t.Fatal(err)
	}
	pages, err := s.ChapterPages(context.Background(), mango.Store{}, mango.Chapter{URL: server.URL + "/chapter"})
	if err != nil || len(pages) != 1 {
		t.Fatalf("got %v, %v", pages, err)
	}
	if err := s.Close(); err != nil || !destroyed {
		t.Errorf("close: %v, session destroyed %t", err, destroyed)
	}
}

func TestFetchHeadless(t *testing.T) {
	if testing.Short() || launcher.NewBrowser().Validate() != nil {
		t.Skip("needs rod's chromium already downloaded")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/chapter":
			// the images are only there once the script ran
			w.Write([]byte(`<html><body><div id="r"></div><script>document.getElementById("r").innerHTML = '<img src="/1.png"><img src="/2.png">'</script></body></html>`))
		case "/":
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	config := &Configuration{
		Name: "test", BaseURL: server.URL, Fetch: fetch.Headless,
		PageExtractor: &PageExtractor{Selector: "img", URL: func(s *goquery.Selection) string { return s.AttrOr("src", "") }},
	}
	s, err := NewScraper(context.Background(), config, mango.DefaultOptions(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	pages, err := s.ChapterPages(context.Background(), mango.Store{}, mango.Chapter{URL: server.URL + "/chapter"})
	if err != nil || len(pages) != 2 || pages[1].(*mango.Page).URL != server.URL+"/2.png" {
		t.Fatalf("got %v, %v", pages, err)
	}
	if _, err := s.ChapterPages(context.Background(), mango.Store{}, mango.Chapter{URL: server.URL + "/gone"}); !errors.Is(err, mango.ErrNotFound) {
		t.Errorf("got %v, want the page's 404", err)
	}
}

func TestSearchPagination(t *testing.T) {
	// /search?page=N has mangas N*10, N*10+1, links to the next one up to page 3; page 4 is a 404
	var requested []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			return
		}
		requested = append(requested, r.URL.Query().Get("page"))
		n, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if n == 0 {
			n = 1
		}
		if n > 3 {
			http.NotFound(w, r)
			return
		}
		ids := []int{n * 10, n*10 + 1}
		if r.URL.Query().Get("repeat") != "" && n > 1 {
			ids = []int{10, 11}
		}
		html := `<html><body>`
		for _, id := range ids {
			html += fmt.Sprintf(`<a class="manga" href="/manga/%d">Manga %d</a>`, id, id)
		}
		if n < 3 {
			q := r.URL.Query()
			q.Set("page", strconv.Itoa(n+1))
			html += fmt.Sprintf(`<div class="next"><a href="?%s">Next</a></div>`, q.Encode())
		}
		w.Write([]byte(html + `</body></html>`))
	}))
	defer server.Close()

	pageURL := func(searchURL string, n int) (string, error) {
		return searchURL + "&page=" + strconv.Itoa(n), nil
	}
	for _, tt := range []struct {
		name       string
		query      string
		pagination *Pagination
		want       int
		requested  string
	}{
		{"none", "a", nil, 2, ""},
		{"next link", "a", &Pagination{NextSelector: "div.next"}, 6, ",2,3"},
		{"url builder", "a", &Pagination{URL: pageURL}, 6, ",2,3,4"},
		{"selector and url", "a", &Pagination{NextSelector: "div.next", URL: pageURL}, 6, ",2,3"},
		{"cap", "a", &Pagination{NextSelector: "div.next", MaxPages: 2}, 4, ",2"},
		{"no new mangas", "a&repeat=1", &Pagination{URL: pageURL}, 2, ",2"},
	} {
		requested = nil
		config := &Configuration{
			Name: "test", BaseURL: server.URL + "/", Parallelism: 1, SearchPagination: tt.pagination,
			GenerateSearchURL: func(base, query string) (string, error) { return base + "search?q=" + query, nil },
			MangaExtractor: &MangaExtractor{
				Selector: "a.manga",
				Title:    func(s *goquery.Selection) string { return s.Text() },
				URL:      func(s *goquery.Selection) string { return s.AttrOr("href", "") },
				ID:       func(u string) string { return strings.Join(PathParts(u), "/") },
				Cover:    func(*goquery.Selection) string { return "" },
			},
		}
		s, err := NewScraper(context.Background(), config, mango.DefaultOptions(), nil)
		if err != nil {
			t.Fatal(err)
		}
		mangas, err := s.SearchMangas(context.Background(), mango.Store{}, tt.query)
		if err != nil || len(mangas) != tt.want {
			t.Errorf("%s: got %d mangas (%v), want %d", tt.name, len(mangas), err, tt.want)
		}
		if got := strings.Join(requested, ","); got != tt.requested {
			t.Errorf("%s: requested pages %q, want %q", tt.name, got, tt.requested)
		}
	}
}
