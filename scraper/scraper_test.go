package scraper

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal/mangadata"
	mango "github.com/luevano/mangoprovider"
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
	if err == nil || !strings.Contains(err.Error(), "GET "+url+": 503 service unavailable") {
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
type hostRecorder struct{ hosts chan string }

func (h hostRecorder) RoundTrip(r *http.Request) (*http.Response, error) {
	h.hosts <- r.Header.Get("Host")
	return http.DefaultTransport.RoundTrip(r)
}

func TestRequestHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	config := &Configuration{Name: "test", BaseURL: server.URL, Parallelism: 1, PageExtractor: &PageExtractor{Selector: "img"}}
	s, err := NewScraper(context.Background(), config, mango.DefaultOptions(), nil)
	if err != nil {
		t.Fatal(err)
	}
	recorder := hostRecorder{make(chan string, 1)}
	s.collector.WithTransport(recorder)
	s.ChapterPages(context.Background(), mango.Store{}, mango.Chapter{URL: server.URL + "/chapter"})

	select {
	case host := <-recorder.hosts:
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
