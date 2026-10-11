package asurascans

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mango "github.com/luevano/mangoprovider"
	"golang.org/x/time/rate"
)

// responses are trimmed copies of the api's (checked 2026-10-11).
var responses = map[string]string{
	"/search":                                        `{"data":[{"slug":"solo-farming-in-the-tower","title":"Solo Farming In The Tower","alt_titles":["Tower Farmer"],"type":"manhwa","status":"ongoing","author":"Lim Hyeong","public_url":"/comics/solo-farming-in-the-tower-bd5bdaf8"}],"meta":{"total":1,"per_page":20,"has_more":false}}`,
	"/series/solo-farming-in-the-tower":              `{"series":{"slug":"solo-farming-in-the-tower","title":"Solo Farming In The Tower","alternative_titles":"나혼자 탑에서 농사 • Tower Farmer","type":"manhwa","status":"ongoing","public_url":"/comics/solo-farming-in-the-tower-bd5bdaf8"}}`,
	"/series/solo-farming-in-the-tower/chapters":     `{"data":[{"number":146,"slug":"chapter-146","is_premium":true,"early_access_until":"2999-01-01T00:00:00Z"},{"number":145,"slug":"chapter-145","is_premium":true,"early_access_until":"2026-10-04T12:36:06.561124Z","published_at":"2026-10-04T06:36:06.561124Z"},{"number":1.5,"slug":"66e058fe","published_at":"2025-12-25T20:41:13Z"}]}`,
	"/series/solo-farming-in-the-tower/chapters/145": `{"data":{"is_locked":false,"chapter":{"pages":[{"url":"https://cdn.asurascans.com/c/145/a.webp?v=1"},{"url":"https://cdn.asurascans.com/c/145/b.jpg?v=1"}]}}}`,
	"/series/solo-farming-in-the-tower/chapters/146": `{"data":{"is_locked":true,"unlock_time":"2026-10-11T09:31:27Z","chapter":{"pages":null}}}`,
}

func newTest(t *testing.T) *asura {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := responses[r.URL.Path]
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return &asura{client: server.Client(), limiter: rate.NewLimiter(rate.Inf, 0), base: server.URL + "/"}
}

func TestSearch(t *testing.T) {
	a := newTest(t)
	mangas, err := a.search(context.Background(), "solo")
	if err != nil || len(mangas) != 1 {
		t.Fatalf("got %v, %v", mangas, err)
	}
	m := mangas[0].(*mango.Manga)
	meta := *m.Metadata_
	if m.ID != "solo-farming-in-the-tower" || m.URL != website+"comics/solo-farming-in-the-tower-bd5bdaf8" || meta.Country() != "KR" || len(meta.AlternateTitles()) != 1 {
		t.Errorf("got %+v, %+v", m, meta)
	}
}

func TestSearchByID(t *testing.T) {
	a := newTest(t)
	mangas, err := a.SearchMangas(context.Background(), mango.Store{}, "mid: solo-farming-in-the-tower")
	if err != nil || len(mangas) != 1 {
		t.Fatalf("got %v, %v", mangas, err)
	}
	if alt := (*mangas[0].(*mango.Manga).Metadata_).AlternateTitles(); len(alt) != 2 || alt[1] != "Tower Farmer" {
		t.Errorf("alt titles %q, want the joined ones split", alt)
	}

	if _, err := a.SearchMangas(context.Background(), mango.Store{}, "mid: nope"); !errors.Is(err, mango.ErrNotFound) {
		t.Errorf("unknown id: %v, want not found", err)
	}
}

func TestChapters(t *testing.T) {
	a := newTest(t)
	manga := &mango.Manga{ID: "solo-farming-in-the-tower", URL: website + "comics/solo-farming-in-the-tower-bd5bdaf8"}
	chapters, err := a.chapters(context.Background(), mango.Volume{Manga_: manga})
	if err != nil || len(chapters) != 2 {
		t.Fatalf("got %v, %v, want 2 without the early access one", chapters, err)
	}
	first := chapters[0].(*mango.Chapter)
	if first.Number != 1.5 || first.ID != "66e058fe" || first.URL != manga.URL+"/chapter/1.5" || first.Date.Year != 2025 {
		t.Errorf("first chapter %+v, want 1.5 (oldest first)", first)
	}

	a.filter.ShowUnavailableChapters = true
	chapters, err = a.chapters(context.Background(), mango.Volume{Manga_: manga})
	if err != nil || len(chapters) != 3 {
		t.Fatalf("got %v, %v, want 3 when showing unavailable ones", chapters, err)
	}
	if c := chapters[2].(*mango.Chapter); c.Number != 146 || c.Unavailable == "" || chapters[1].(*mango.Chapter).Unavailable != "" {
		t.Errorf("got %+v, want only 146 unavailable", chapters)
	}
}

func TestThrottle(t *testing.T) {
	a := newTest(t)
	a.limiter = rate.NewLimiter(rate.Every(100*time.Millisecond), 2)
	start := time.Now()
	for range 4 {
		if _, err := a.search(context.Background(), "solo"); err != nil {
			t.Fatal(err)
		}
	}
	// 2 right away, then one every 100ms
	if took := time.Since(start); took < 200*time.Millisecond {
		t.Errorf("4 requests took %v, want at least 200ms", took)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := a.search(ctx, "solo"); err == nil {
		t.Error("cancelled while throttled: no error")
	}
}

func TestChapterPages(t *testing.T) {
	a := newTest(t)
	volume := &mango.Volume{Manga_: &mango.Manga{ID: "solo-farming-in-the-tower"}}
	pages, err := a.ChapterPages(context.Background(), mango.Store{}, mango.Chapter{ID: "145", Volume_: volume})
	if err != nil || len(pages) != 2 {
		t.Fatalf("got %v, %v", pages, err)
	}
	if p := pages[0].(*mango.Page); p.Ext != ".webp" {
		t.Errorf("ext %q, want .webp without the query", p.Ext)
	}

	if _, err := a.ChapterPages(context.Background(), mango.Store{}, mango.Chapter{ID: "146", Volume_: volume}); err == nil {
		t.Error("locked chapter: no error")
	}
}
