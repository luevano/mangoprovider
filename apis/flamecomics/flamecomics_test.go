package flamecomics

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	mango "github.com/luevano/mangoprovider"
)

// pages are trimmed copies of what the site embeds (checked 2026-10-10).
var pages = map[string]string{
	"/series/2":     `{"series":{"series_id":2,"title":"Omniscient Reader's Viewpoint","altTitles":[" ORV ",""],"type":"Manhwa","status":"Ongoing","year":2020,"cover":"thumbnail.png","author":["Sing N Song"]},"chapters":[{"chapter":"1.00","title":"One","token":"bbb","release_date":1611705600},{"chapter":"0.00","title":"Prologue","token":"aaa","release_date":1611705600}]}`,
	"/series/2/aaa": `{"chapter":{"token":"aaa","images":{"10":{"name":"p10.jpg"},"2":{"name":"p02.jpg"},"0":{"name":"p00.png"}}}}`,
}

func newTest(t *testing.T) *flame {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestSearchByID(t *testing.T) {
	f := newTest(t)
	mangas, err := f.SearchMangas(context.Background(), mango.Store{}, "mid: 2")
	if err != nil || len(mangas) != 1 {
		t.Fatalf("got %v, %v", mangas, err)
	}
	m := mangas[0].(*mango.Manga)
	meta := *m.Metadata_
	if m.ID != "2" || m.Cover != cdn+"2/thumbnail.png" || meta.Title() != "Omniscient Reader's Viewpoint" || len(meta.AlternateTitles()) != 1 {
		t.Errorf("got %+v, %+v", m, meta)
	}

	if mangas, err := f.SearchMangas(context.Background(), mango.Store{}, "mid: 999"); err != nil || len(mangas) != 0 {
		t.Errorf("unknown id: %v, %v, want no mangas", mangas, err)
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
