package mangapluscreators

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/luevano/mangoplus/creators"
	mango "github.com/luevano/mangoprovider"
)

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// newTest: The provider with the api requests going to handler instead.
func newTest(t *testing.T, handler http.HandlerFunc) *mpc {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	target, _ := url.Parse(server.URL)

	o := creators.DefaultOptions()
	o.HTTPClient = &http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
		r.URL.Scheme = target.Scheme
		r.URL.Host = target.Host
		return http.DefaultTransport.RoundTrip(r)
	})}
	client, err := creators.NewCreatorsClient(o)
	if err != nil {
		t.Fatal(err)
	}
	return &mpc{client: client, filter: mango.DefaultFilterOptions(), options: mango.MangaPlusCreatorsOptions{Options: o}}
}

// api: Answers each path (with its page query) with the json given.
func api(t *testing.T, answers map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path
		if page := r.URL.Query().Get("page"); page != "" {
			key += "?page=" + page
		}
		body, ok := answers[key]
		if !ok {
			t.Errorf("unexpected request %s", key)
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(body))
	}
}

func TestSearch(t *testing.T) {
	c := newTest(t, api(t, map[string]string{
		"/api/mpc/search/titles?page=1": `{"result":"OK","mpcTitlesDto":{"titleList":[{"titleId":"fm1","title":"Rauch","handleName":"A","firstPublishDate":1681000000000}],"pagination":{"page":1,"maxPage":2}}}`,
		"/api/mpc/search/titles?page=2": `{"result":"OK","mpcTitlesDto":{"titleList":[{"titleId":"fm2","title":"Rauch 2"}],"pagination":{"page":2,"maxPage":2}}}`,
	}))
	mangas, err := c.SearchMangas(context.Background(), mango.Store{}, "rauch")
	if err != nil || len(mangas) != 2 {
		t.Fatalf("got %v, %v", mangas, err)
	}
	m := mangas[0].(*mango.Manga)
	meta := *m.Metadata_
	if m.ID != "fm1" || m.Title != "Rauch" || m.URL != website+"titles/fm1" || meta.StartDate().Year != 2023 {
		t.Errorf("got %+v, %+v", m, meta)
	}
}

func TestSearchEmpty(t *testing.T) {
	for _, body := range []string{
		`{"result":"OK","mpcTitlesDto":{"titleList":[],"pagination":{"page":1,"maxPage":0}}}`,
		`{"result":"OK","mpcTitlesDto":{}}`,
		`{"result":"OK"}`,
	} {
		c := newTest(t, api(t, map[string]string{"/api/mpc/search/titles?page=1": body}))
		mangas, err := c.SearchMangas(context.Background(), mango.Store{}, "nothing")
		if err != nil || len(mangas) != 0 {
			t.Errorf("%s: wanted no results, got %v, %v", body, mangas, err)
		}
	}
}

func TestVolumes(t *testing.T) {
	c := newTest(t, api(t, nil))
	volumes, err := c.MangaVolumes(context.Background(), mango.Store{}, mango.Manga{ID: "fm1"})
	if err != nil || len(volumes) != 1 || !volumes[0].(*mango.Volume).None {
		t.Errorf("wanted the \"no volumes\" volume, got %v, %v", volumes, err)
	}
}

func TestChapters(t *testing.T) {
	c := newTest(t, api(t, map[string]string{
		"/api/mpc/titles/fm1/episodes/?page=1": `{"result":"OK","mpcEpisodesDto":{"episodeList":[{"episodeId":"ep1","episodeTitle":"Start","numbering":1,"firstPublishDate":1681000000000},{"episodeId":"ep2","episodeTitle":"Next","numbering":2}],"pagination":{"page":1,"maxPage":1}}}`,
		"/api/mpc/titles/fm2/episodes/?page=1": `{"result":"OK","mpcEpisodesDto":{"episodeList":[],"pagination":{"page":1,"maxPage":0}}}`,
	}))
	volume := mango.Volume{Number: 1, Manga_: &mango.Manga{ID: "fm1"}}
	chapters, err := c.VolumeChapters(context.Background(), mango.Store{}, volume)
	if err != nil || len(chapters) != 2 {
		t.Fatalf("got %v, %v", chapters, err)
	}
	first, second := chapters[0].(*mango.Chapter), chapters[1].(*mango.Chapter)
	if first.ID != "ep1" || first.Number != 1 || first.Date.Year != 2023 || first.URL != website+"episodes/ep1/" {
		t.Errorf("got %+v", first)
	}
	// no date isn't 1970
	if second.Date.Year != 0 {
		t.Errorf("wanted no date, got %v", second.Date)
	}

	volume.Manga_ = &mango.Manga{ID: "fm2"}
	chapters, err = c.VolumeChapters(context.Background(), mango.Store{}, volume)
	if err != nil || len(chapters) != 0 {
		t.Errorf("wanted no chapters, got %v, %v", chapters, err)
	}
}

func TestChapterPages(t *testing.T) {
	c := newTest(t, api(t, map[string]string{
		"/api/mpc/episodes/pageList/ep1/": `{"result":"OK","pageList":[{"pageNo":1,"publicBgImage":"https://cdn.test/1.jpg?sig=x"}]}`,
	}))
	pages, err := c.ChapterPages(context.Background(), mango.Store{}, mango.Chapter{ID: "ep1", URL: website + "episodes/ep1/"})
	if err != nil || len(pages) != 1 {
		t.Fatalf("got %v, %v", pages, err)
	}
	if p := pages[0].(*mango.Page); p.URL != "https://cdn.test/1.jpg?sig=x" || p.Ext != ".jpg" || p.Headers["Referer"] != website+"episodes/ep1/" {
		t.Errorf("got %+v", p)
	}
}

func TestWAF(t *testing.T) {
	c := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-amzn-waf-action", "challenge")
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`<html><script src="https://x.token.awswaf.com/challenge.js"></script></html>`))
	})
	_, err := c.SearchMangas(context.Background(), mango.Store{}, "rauch")
	if err == nil || !strings.Contains(err.Error(), "blocked by AWS WAF") {
		t.Errorf("wanted the waf error, got %v", err)
	}
}
