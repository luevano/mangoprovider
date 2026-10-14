// Package scrapertest runs a scraper against html fixtures served locally.
package scrapertest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
)

// New serves files (url path -> file in testdata/) and returns the scraper for config with its
// BaseURL set to the server's (ends with "/"). A key with a query ("/search?page=2") is
// only for requests with those params (in the url or a POSTed form), and wins over its path
// alone. Any other path is a test error and a 404.
func New(t *testing.T, config *scraper.Configuration, files map[string]string) (*scraper.Scraper, string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		file, ok := match(files, r.URL.Path, r.Form)
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
	s, err := scraper.NewScraper(context.Background(), config, Options(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return s, config.BaseURL
}

// Options are mango.DefaultOptions that reach the local test servers.
func Options() mango.Options {
	o := mango.DefaultOptions()
	o.AllowPrivateAddresses = true
	return o
}

// match: The file for the request, the key with the most matching params.
func match(files map[string]string, path string, params url.Values) (string, bool) {
	file, ok := files[path]
	best := 0
	for key, f := range files {
		keyPath, rawQuery, hasQuery := strings.Cut(key, "?")
		if !hasQuery || keyPath != path {
			continue
		}
		want, err := url.ParseQuery(rawQuery)
		if err != nil {
			continue
		}
		matches := true
		for k := range want {
			if params.Get(k) != want.Get(k) {
				matches = false
			}
		}
		if matches && len(want) > best {
			file, ok, best = f, true, len(want)
		}
	}
	return file, ok
}
