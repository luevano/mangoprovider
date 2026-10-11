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
	for _, id := range []string{"3/nope", "4/gone"} {
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
