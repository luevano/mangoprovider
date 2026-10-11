package mangoprovider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/mangadata"
)

var errAPI = errors.New("mangodex: GET /manga: 429 too many requests")

func TestProviderErrorContext(t *testing.T) {
	manga := &Manga{Title: "One Piece"}
	volume := &Volume{Number: 1, Manga_: manga}
	chapter := &Chapter{Title: "Romance Dawn", Volume_: volume}
	p := &Provider{
		ProviderInfo: libmangal.ProviderInfo{ID: "mango-test"},
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

func TestGenericGetPageImageStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	_, err := GenericGetPageImage(context.Background(), server.Client(), Page{URL: server.URL + "/1.png"})
	want := "GET " + server.URL + "/1.png: 404 not found"
	if err == nil || err.Error() != want {
		t.Errorf("wanted %q, got %v", want, err)
	}
}

func TestAccount(t *testing.T) {
	p := &Provider{ProviderInfo: libmangal.ProviderInfo{ID: "mango-x"}}
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
