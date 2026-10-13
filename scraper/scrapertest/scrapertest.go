// Package scrapertest runs a scraper against html fixtures served locally.
package scrapertest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
)

// New serves files (url path -> file in testdata/) and returns the scraper for config with its
// BaseURL set to the server's (ends with "/"). Any other path is a test error and a 404.
func New(t *testing.T, config *scraper.Configuration, files map[string]string) (*scraper.Scraper, string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			return // NewScraper's redirect check
		}
		file, ok := files[r.URL.Path]
		if !ok {
			t.Errorf("unexpected request %s", r.URL)
			http.NotFound(w, r)
			return
		}
		html, err := os.ReadFile(filepath.Join("testdata", file))
		if err != nil {
			t.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(html)
	}))
	t.Cleanup(server.Close)

	config.BaseURL = server.URL + "/"
	s, err := scraper.NewScraper(context.Background(), config, mango.DefaultOptions(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return s, config.BaseURL
}
