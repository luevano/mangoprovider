package mangoprovider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/progress"
)

var errAPI = errors.New("mangodex: GET /manga: 429 too many requests")

func TestProviderErrorContext(t *testing.T) {
	manga := &Manga{Title: "One Piece"}
	volume := &Volume{Number: 1, Manga_: manga}
	chapter := &Chapter{Title: "Romance Dawn", Volume_: volume}
	p := &Provider{
		ID: "mango-test",
		F: Functions{
			SearchMangas: func(context.Context, Store, string) ([]mangadata.Manga, error) { return nil, errAPI },
			MangaVolumes: func(context.Context, Store, Manga) ([]mangadata.Volume, error) { return nil, errAPI },
			VolumeChapters: func(context.Context, Store, Volume) ([]mangadata.Chapter, error) {
				return nil, errAPI
			},
			ChapterPages: func(context.Context, Store, Chapter) ([]mangadata.Page, error) { return nil, errAPI },
			GetPageImage: func(context.Context, *http.Client, Page) ([]byte, error) { return nil, errAPI },
		},
	}
	ctx := context.Background()

	_, searchErr := p.SearchMangas(ctx, "one piece")
	_, volumesErr := p.MangaVolumes(ctx, manga)
	_, chaptersErr := p.VolumeChapters(ctx, volume)
	_, pagesErr := p.ChapterPages(ctx, chapter)
	_, imageErr := p.GetPageImage(ctx, &Page{Chapter_: chapter})
	tests := []struct {
		err  error
		want string
	}{
		{searchErr, `mango-test: search "one piece": ` + errAPI.Error()},
		{volumesErr, `mango-test: volumes of "One Piece": ` + errAPI.Error()},
		{chaptersErr, `mango-test: chapters of "One Piece" vol 1: ` + errAPI.Error()},
		{pagesErr, `mango-test: pages of chapter "Romance Dawn": ` + errAPI.Error()},
		{imageErr, `mango-test: image for chapter "Romance Dawn": ` + errAPI.Error()},
	}
	for _, tt := range tests {
		if tt.err == nil || tt.err.Error() != tt.want {
			t.Errorf("wanted %q, got %v", tt.want, tt.err)
		}
		if !errors.Is(tt.err, errAPI) {
			t.Errorf("%v should still wrap the original error", tt.err)
		}
	}
}

func TestAccount(t *testing.T) {
	p := &Provider{ID: "mango-x"}
	if _, found, err := p.Account(context.Background()); found || err != nil {
		t.Errorf("found %t, %v without an Account function", found, err)
	}

	p.F.Account = func(context.Context) (libmangal.Account, bool, error) {
		return libmangal.Account{Status: "deluxe"}, true, nil
	}
	if account, found, err := p.Account(context.Background()); !found || err != nil || account.Status != "deluxe" {
		t.Errorf("got %+v %t %v", account, found, err)
	}

	p.F.Account = func(context.Context) (libmangal.Account, bool, error) {
		return libmangal.Account{}, false, errors.New("down")
	}
	if _, _, err := p.Account(context.Background()); err == nil || err.Error() != "mango-x: account: down" {
		t.Errorf("got %v", err)
	}
}

func TestDefaultCacheHits(t *testing.T) {
	searches := 0
	loader := &Loader{
		ID: "mango-test", Name: "Test", Version: "0.1.0",
		Options: DefaultOptions(),
		F: func(context.Context, *Logger) (Functions, error) {
			return Functions{
				SearchMangas: func(_ context.Context, store Store, query string) ([]mangadata.Manga, error) {
					var mangas []mangadata.Manga
					if found, err := store.GetMangas(query, query, &mangas); err != nil || found {
						return mangas, err
					}
					searches++
					mangas = []mangadata.Manga{&Manga{Title: "One Piece"}}
					return mangas, store.SetMangas(query, mangas)
				},
			}, nil
		},
	}
	p, err := loader.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	for range 2 {
		if mangas, err := p.SearchMangas(context.Background(), "one piece"); err != nil || len(mangas) != 1 {
			t.Fatalf("got %v, %v", mangas, err)
		}
	}
	if searches != 1 {
		t.Errorf("searched %d times, want 1 (second from the cache)", searches)
	}
}

func TestMangaByID(t *testing.T) {
	want := &Manga{Title: "One Piece", ID: "13"}
	p := &Provider{
		ID: "mango-test",
		F: Functions{
			MangaByID: func(_ context.Context, _ Store, id string) (mangadata.Manga, bool, error) {
				switch id {
				case "13":
					return want, true, nil
				case "fail":
					return nil, false, errAPI
				}
				return nil, false, nil
			},
		},
	}
	ctx := context.Background()

	if manga, found, err := p.MangaByID(ctx, "13"); err != nil || !found || manga != want {
		t.Errorf("got %v, %v, %v", manga, found, err)
	}
	if manga, found, err := p.MangaByID(ctx, "nope"); err != nil || found || manga != nil {
		t.Errorf("unknown id: %v, %v, %v", manga, found, err)
	}
	if _, _, err := p.MangaByID(ctx, "fail"); !errors.Is(err, errAPI) || err.Error() != `mango-test: manga "fail": `+errAPI.Error() {
		t.Errorf("error: %v", err)
	}

	p.F.MangaByID = nil
	if _, _, err := p.MangaByID(ctx, "13"); !errors.Is(err, libmangal.ErrMangaByIDUnsupported) {
		t.Errorf("without MangaByID: %v, want ErrMangaByIDUnsupported", err)
	}
}

func TestGetPageImageReportsRetries(t *testing.T) {
	defer func(w time.Duration) { pageRetryWait = w }(pageRetryWait)
	pageRetryWait = time.Millisecond
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte("png"))
	}))
	defer server.Close()

	info := libmangal.ProviderInfo{ID: "mango-test", Name: "Test"}
	p := &Provider{ProviderInfo: info, client: server.Client(), log: NewProviderLogger(info)}
	var got []string
	ctx := progress.WithReporter(context.Background(), func(s progress.Status) {
		if s.Source != "mango-test" {
			t.Errorf("source %q", s.Source)
		}
		got = append(got, s.String())
	})
	chapter := &Chapter{Title: "Ch. 40"}
	if _, err := p.GetPageImage(ctx, &Page{URL: server.URL + "/1.png", Chapter_: chapter}); err != nil {
		t.Fatal(err)
	}
	want := `Test: page of chapter "Ch. 40": 429 too many requests, retry 1/3 in 0s`
	if len(got) != 1 || got[0] != want {
		t.Errorf("reported %q, want %q", got, want)
	}
}
