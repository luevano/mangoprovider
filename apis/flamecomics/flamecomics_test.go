package flamecomics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	mango "github.com/luevano/mangoprovider"
	"golang.org/x/time/rate"
)

// pages are trimmed copies of what the site embeds (checked 2026-10-10).
var pages = map[string]string{
	"/browse":       `{"series":[{"title":"Omniscient Reader"},{"series_id":2,"title":"Omniscient Reader's Viewpoint","type":"Manhwa"}]}`,
	"/series/2":     `{"series":{"series_id":2,"title":"Omniscient Reader's Viewpoint","altTitles":[" ORV ",""],"type":"Manhwa","status":"Ongoing","year":2020,"cover":"thumbnail.png","author":["Sing N Song"]},"chapters":[{"chapter":"1.00","title":"One","token":"bbb","release_date":1611705600},{"chapter":"0.00","title":"Prologue","token":"aaa","release_date":1611705600}]}`,
	"/series/2/aaa": `{"chapter":{"token":"aaa","images":{"10":{"name":"p10.jpg"},"2":{"name":"p02.jpg"},"0":{"name":"p00.png"}}}}`,
}

func newTest(t *testing.T) *flame {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/limited":
			w.WriteHeader(http.StatusTooManyRequests)
			return
		case "/down":
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		props, ok := pages[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, `<html><script id="__NEXT_DATA__" type="application/json">{"props":{"pageProps":%s}}</script></html>`, props)
	}))
	t.Cleanup(server.Close)
	return &flame{client: server.Client(), base: server.URL + "/"}
}

func TestMangaByIDMetadata(t *testing.T) {
	f := newTest(t)
	manga, found, err := f.MangaByID(context.Background(), mango.Store{}, "2")
	if err != nil || !found {
		t.Fatalf("got %v, %v, %v", manga, found, err)
	}
	m := manga.(*mango.Manga)
	meta := *m.Metadata_
	if m.ID != "2" || m.Cover != cdn+"2/thumbnail.png" || meta.Title() != "Omniscient Reader's Viewpoint" || len(meta.AlternateTitles()) != 1 {
		t.Errorf("got %+v, %+v", m, meta)
	}

	if manga, found, err := f.MangaByID(context.Background(), mango.Store{}, "999"); err != nil || found {
		t.Errorf("unknown id: %v, %v, %v, want not found", manga, found, err)
	}
}

func TestMangaByID(t *testing.T) {
	pages["/series/3"] = `{"series":null}`
	defer delete(pages, "/series/3")
	f := newTest(t)
	manga, found, err := f.MangaByID(context.Background(), mango.Store{}, "2")
	if err != nil || !found || manga.Info().ID != "2" {
		t.Fatalf("got %v, %v, %v", manga, found, err)
	}
	for _, id := range []string{"999", "3"} {
		if manga, found, err := f.MangaByID(context.Background(), mango.Store{}, id); err != nil || found {
			t.Errorf("unknown id %s: %v, %v, %v, want not found", id, manga, found, err)
		}
	}
}

func TestChapterPages(t *testing.T) {
	f := newTest(t)
	chapter := mango.Chapter{ID: "aaa", Volume_: &mango.Volume{Manga_: &mango.Manga{ID: "2"}}}
	got, err := f.ChapterPages(context.Background(), mango.Store{}, chapter)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"p00.png", "p02.jpg", "p10.jpg"}
	if len(got) != len(want) {
		t.Fatalf("got %d pages, want %d", len(got), len(want))
	}
	for i, name := range want {
		if url := got[i].(*mango.Page).URL; url != cdn+"2/aaa/"+name {
			t.Errorf("page %d: %s, want %s", i, url, name)
		}
	}
}

func TestPageStatusErrors(t *testing.T) {
	f := newTest(t)
	var v any
	if err := f.page(context.Background(), "limited", &v); !errors.Is(err, mango.ErrRateLimited) {
		t.Errorf("429: %v", err)
	}
	if err := f.page(context.Background(), "down", &v); !errors.Is(err, mango.ErrSiteDown) {
		t.Errorf("502: %v", err)
	}
}

func TestThrottle(t *testing.T) {
	f := newTest(t)
	f.limiter = rate.NewLimiter(rate.Every(100*time.Millisecond), 2)
	f.client = newClient(f.client, f.limiter, nil)
	start := time.Now()
	for range 4 {
		if _, _, err := f.MangaByID(context.Background(), mango.Store{}, "2"); err != nil {
			t.Fatal(err)
		}
	}
	// 2 right away, then one every 100ms
	if took := time.Since(start); took < 200*time.Millisecond {
		t.Errorf("4 requests took %v, want at least 200ms", took)
	}
}

func TestVolumeChapters(t *testing.T) {
	// newest first, like the site
	pages["/series/4"] = `{"chapters":[{"chapter":"2.00","title":"Two","token":"c"},{"chapter":"","title":"Extra","token":"x"},{"chapter":"1.00","title":"One","token":"b","release_date":1611705600}]}`
	defer delete(pages, "/series/4")
	f := newTest(t)
	volume := mango.Volume{Number: 1, None: true, Manga_: &mango.Manga{ID: "4"}}
	chapters, err := f.VolumeChapters(context.Background(), mango.Store{}, volume)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		id     string
		number float32
		date   string
	}{{"b", 1, "2021-01-27"}, {"x", 1.5, ""}, {"c", 2, ""}}
	if len(chapters) != len(want) {
		t.Fatalf("got %v, want %d chapters", chapters, len(want))
	}
	for i, w := range want {
		c := chapters[i].(*mango.Chapter)
		if c.ID != w.id || c.Number != w.number || c.Date.String() != w.date {
			t.Errorf("chapter %d: got %s %v %q, want %s %v %q", i, c.ID, c.Number, c.Date, w.id, w.number, w.date)
		}
	}
}

func TestLimit(t *testing.T) {
	// keiyoushi's rateLimit(2, 2.seconds)
	if l := newLimiter(); l.Limit() != 1 || l.Burst() != 2 {
		t.Errorf("got %v/s burst %d, want 1/s burst 2", l.Limit(), l.Burst())
	}
}

func TestPageImageThrottled(t *testing.T) {
	image := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("png")) }))
	defer image.Close()
	f := newTest(t)
	f.limiter = rate.NewLimiter(rate.Every(100*time.Millisecond), 1)
	start := time.Now()
	for range 3 {
		if _, err := f.GetPageImage(context.Background(), image.Client(), mango.Page{URL: image.URL + "/p.png"}); err != nil {
			t.Fatal(err)
		}
	}
	if took := time.Since(start); took < 200*time.Millisecond {
		t.Errorf("3 images took %v, want at least 200ms", took)
	}
}

func TestClientRetries(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, `<script id="__NEXT_DATA__" type="application/json">{"props":{"pageProps":{"ok":true}}}</script>`)
	}))
	defer server.Close()
	f := &flame{client: newClient(server.Client(), rate.NewLimiter(rate.Inf, 0), nil), base: server.URL + "/"}
	var v struct{ OK bool }
	if err := f.page(context.Background(), "x", &v); err != nil || !v.OK || hits.Load() != 2 {
		t.Errorf("got %v %v after %d requests", v, err, hits.Load())
	}
}

func TestChapterNumber(t *testing.T) {
	for s, want := range map[string]bool{"10": true, "10.5": true, "0": true, "": false, "NaN": false, "Inf": false, "-Inf": false, "10a": false} {
		if _, ok := chapterNumber(s); ok != want {
			t.Errorf("%q: wanted %v, got %v", s, want, ok)
		}
	}
}

func TestSearchSkipsWithoutID(t *testing.T) {
	mangas, err := newTest(t).SearchMangas(context.Background(), mango.Store{}, "omniscient")
	if err != nil || len(mangas) != 1 || mangas[0].Info().ID != "2" {
		t.Errorf("got %v, %v, want only series 2", mangas, err)
	}
}
