package mangadex

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/progress"
	"github.com/luevano/mangodex"
	mango "github.com/luevano/mangoprovider"
)

// fakeDex: Transport answering like MangaDex, its MangaDex@Home nodes and the report endpoint,
// counting the requests by path.
type fakeDex struct {
	mu       sync.Mutex
	requests map[string]int
	// nodeStatus: How the nodes answer the images.
	nodeStatus int
	// total: How many mangas the search says match.
	total int
	// queries: The last query by path.
	queries map[string]string
	// chapters: Optional, the /chapter answer's data.
	chapters string
}

func newFakeDex() *fakeDex {
	return &fakeDex{requests: map[string]int{}, queries: map[string]string{}, nodeStatus: http.StatusOK, total: 10000}
}

func (f *fakeDex) count(path string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests[path]
}

func (f *fakeDex) RoundTrip(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	f.requests[r.URL.Path]++
	f.queries[r.URL.Path] = r.URL.RawQuery
	nodeStatus, total := f.nodeStatus, f.total
	f.mu.Unlock()

	w := httptest.NewRecorder()
	switch {
	case r.URL.Path == "/manga":
		// every page is full, as for a query matching thousands
		mangas := make([]string, 100)
		for i := range mangas {
			mangas[i] = `{"id":"m","type":"manga","attributes":{"title":{"en":"A"}}}`
		}
		w.WriteString(`{"result":"ok","data":[` + strings.Join(mangas, ",") + `],"total":` + strconv.Itoa(total) + `}`)
	case r.URL.Path == "/manga/empty":
		w.WriteString(`{"result":"ok","data":null}`)
	case r.URL.Path == "/chapter" && f.chapters != "":
		w.WriteString(`{"result":"ok","data":[` + f.chapters + `]}`)
	case strings.HasPrefix(r.URL.Path, "/at-home/server/"):
		w.WriteString(`{"result":"ok","baseUrl":"https://node.test","chapter":{"hash":"h","data":["1.png","2.png"]}}`)
	case r.URL.Host == "node.test":
		w.WriteHeader(nodeStatus)
		w.WriteString("image")
	default:
		w.WriteHeader(http.StatusOK)
	}
	resp := w.Result()
	resp.Request = r
	return resp, nil
}

func fakeLoader(fake *fakeDex) *mango.Loader {
	options := mango.DefaultOptions()
	options.HTTPClient = &http.Client{Transport: fake}
	return Loader(options).(*mango.Loader)
}

func loadFake(t *testing.T, fake *fakeDex, log *mango.Logger) mango.Functions {
	t.Helper()
	return load(t, fakeLoader(fake), log)
}

func load(t *testing.T, loader *mango.Loader, log *mango.Logger) mango.Functions {
	t.Helper()
	fns, err := loader.F(context.Background(), log)
	if err != nil {
		t.Fatal(err)
	}
	return fns
}

func TestSearchPagesCapped(t *testing.T) {
	fake := newFakeDex()
	fns := loadFake(t, fake, nil)
	mangas, err := fns.SearchMangas(context.Background(), mango.Store{}, "a")
	if err != nil {
		t.Fatal(err)
	}
	if got := fake.count("/manga"); got != searchPages || len(mangas) != searchPages*100 {
		t.Errorf("wanted %d pages, got %d requests, %d mangas", searchPages, got, len(mangas))
	}
}

func TestSearchStopsAtTotal(t *testing.T) {
	fake := newFakeDex()
	fake.total = 200
	fns := loadFake(t, fake, nil)
	mangas, err := fns.SearchMangas(context.Background(), mango.Store{}, "a")
	if err != nil {
		t.Fatal(err)
	}
	if got := fake.count("/manga"); got != 2 || len(mangas) != 200 {
		t.Errorf("wanted 2 pages, got %d requests, %d mangas", got, len(mangas))
	}
}

// chapterPages: The pages of a chapter, sharing it like the ones ChapterPages returns.
func chapterPages(t *testing.T, fns mango.Functions) []mango.Page {
	t.Helper()
	pages, err := fns.ChapterPages(context.Background(), mango.Store{}, mango.Chapter{ID: "c"})
	if err != nil {
		t.Fatal(err)
	}
	var out []mango.Page
	for _, p := range pages {
		out = append(out, *p.(*mango.Page))
	}
	return out
}

func TestAtHomeRefreshedOnce(t *testing.T) {
	fake := newFakeDex()
	fns := loadFake(t, fake, nil)
	pages := chapterPages(t, fns)

	fake.nodeStatus = http.StatusForbidden // the url expired, and the new one fails too
	var wg sync.WaitGroup
	for range 4 {
		for _, page := range pages {
			wg.Go(func() {
				if _, err := fns.GetPageImage(context.Background(), nil, page); err == nil {
					t.Error("wanted an error")
				}
			})
		}
	}
	wg.Wait()
	// one for ChapterPages, one refresh
	if got := fake.count("/at-home/server/c"); got != 2 {
		t.Errorf("wanted the server fetched again once, got %d requests", got)
	}
}

func TestAtHomeRefreshUsed(t *testing.T) {
	fake := newFakeDex()
	fns := loadFake(t, fake, nil)
	pages := chapterPages(t, fns)
	expired := pages[0].Chapter_.AtHome

	fake.nodeStatus = http.StatusForbidden
	fns.GetPageImage(context.Background(), nil, pages[0])
	if pages[0].Chapter_.AtHome == expired {
		t.Error("the new server should be kept for the other pages")
	}
}

func TestAtHomeRateLimited(t *testing.T) {
	fake := newFakeDex()
	fns := loadFake(t, fake, nil)
	pages := chapterPages(t, fns)

	fake.nodeStatus = http.StatusTooManyRequests
	_, err := fns.GetPageImage(context.Background(), nil, pages[0])
	if !errors.Is(err, mangodex.ErrRateLimited) {
		t.Errorf("wanted the rate limit, got %v", err)
	}
	if got := fake.count("/at-home/server/c"); got != 1 {
		t.Errorf("a rate limit shouldn't get a new server, got %d requests", got)
	}
}

func TestLoadsIndependent(t *testing.T) {
	loader := fakeLoader(newFakeDex())
	first := load(t, loader, mango.NewProviderLogger(libmangal.ProviderInfo{ID: "first"}))
	load(t, loader, mango.NewProviderLogger(libmangal.ProviderInfo{ID: "second"}))

	var sources []string
	ctx := progress.WithReporter(context.Background(), func(s progress.Status) {
		sources = append(sources, s.Source)
	})
	if _, err := first.SearchMangas(ctx, mango.Store{}, "a"); err != nil {
		t.Fatal(err)
	}
	if len(sources) == 0 {
		t.Fatal("no progress reported")
	}
	for _, source := range sources {
		if source != "first" {
			t.Errorf("the first provider reported as %q", source)
		}
	}
}

// an empty language is any language: no translatedLanguage[] (it would match none)
func TestAnyLanguage(t *testing.T) {
	for lang, want := range map[string]string{"": "", "es-la": "es-la"} {
		fake := newFakeDex()
		options := mango.DefaultOptions()
		options.HTTPClient = &http.Client{Transport: fake}
		options.Filter.Language = lang
		fns := load(t, Loader(options).(*mango.Loader), nil)
		manga := mango.Manga{ID: "m"}
		fns.MangaVolumes(context.Background(), mango.Store{}, manga)
		fns.VolumeChapters(context.Background(), mango.Store{}, mango.Volume{Number: 1, Manga_: &manga})
		for _, path := range []string{"/manga/m/aggregate", "/chapter"} {
			query, err := url.ParseQuery(fake.queries[path])
			if err != nil {
				t.Fatal(err)
			}
			if got, ok := query["translatedLanguage[]"]; (want == "") == ok || (ok && got[0] != want) {
				t.Errorf("language %q, %s: got %s", lang, path, fake.queries[path])
			}
		}
	}
}

func TestGroups(t *testing.T) {
	chapter := func(id, number, groupID, group string) string {
		return `{"id":"` + id + `","type":"chapter","attributes":{"chapter":"` + number + `","pages":3,"publishAt":"2024-01-02T00:00:00+00:00"},` +
			`"relationships":[{"id":"` + groupID + `","type":"scanlation_group","attributes":{"name":"` + group + `"}}]}`
	}
	const a, b, c = "00000000-0000-0000-0000-00000000000a", "00000000-0000-0000-0000-00000000000b", "00000000-0000-0000-0000-00000000000c"
	fake := newFakeDex()
	fake.chapters = strings.Join([]string{
		chapter("1a", "1", a, "Alpha"), chapter("1b", "1", b, "Beta"), chapter("1c", "1", c, "Gamma"),
		chapter("2b", "2", b, "Beta"), chapter("2c", "2", c, "Gamma"),
		chapter("3c", "3", c, "Gamma"),
	}, ",")
	options := mango.DefaultOptions()
	options.HTTPClient = &http.Client{Transport: fake}
	options.Filter.AvoidDuplicateChapters = true
	options.MangaDex.ExcludeGroups = []string{"gamma"}
	options.MangaDex.PreferGroups = []string{b, "Alpha"}
	fns := load(t, Loader(options).(*mango.Loader), nil)
	manga := mango.Manga{ID: "m"}
	chapters, err := fns.VolumeChapters(context.Background(), mango.Store{}, mango.Volume{Number: 1, Manga_: &manga})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range chapters {
		got = append(got, c.(*mango.Chapter).ID)
	}
	// gamma's left out, beta (by id) preferred over alpha
	if want := []string{"1b", "2b"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("wanted %v, got %v", want, got)
	}
}

func TestChaptersPagesCapped(t *testing.T) {
	defer func(m int) { maxResults = m }(maxResults)
	maxResults = 500
	fake := newFakeDex()
	chapters := make([]string, 100)
	for i := range chapters {
		chapters[i] = `{"id":"00000000-0000-0000-0000-` + strings.Repeat("0", 9) + strconv.Itoa(100+i) + `","type":"chapter","attributes":{"chapter":"` + strconv.Itoa(i) + `","pages":3}}`
	}
	fake.chapters = strings.Join(chapters, ",")
	fns := loadFake(t, fake, nil)
	manga := mango.Manga{ID: "m"}
	if _, err := fns.VolumeChapters(context.Background(), mango.Store{}, mango.Volume{Number: 1, Manga_: &manga}); err != nil {
		t.Fatal(err)
	}
	query, _ := url.ParseQuery(fake.queries["/chapter"])
	if got := fake.count("/chapter"); got != 5 || query.Get("offset") != "400" {
		t.Errorf("wanted 5 pages up to offset 400, got %d, last offset %s", got, query.Get("offset"))
	}
}

func TestAtHomeRefreshedForgotten(t *testing.T) {
	fake := newFakeDex()
	options := mangodex.DefaultOptions()
	options.HTTPClient = &http.Client{Transport: fake}
	client, err := mangodex.NewDexClient(options)
	if err != nil {
		t.Fatal(err)
	}
	d := &dex{client: client, atHomeRefreshed: map[string]*mangodex.AtHomeServer{"c": nil}}
	if _, err := d.ChapterPages(context.Background(), mango.Store{}, mango.Chapter{ID: "c"}); err != nil {
		t.Fatal(err)
	}
	if len(d.atHomeRefreshed) != 0 {
		t.Errorf("wanted the chapter forgotten, got %v", d.atHomeRefreshed)
	}
}

func TestRetiredChapters(t *testing.T) {
	for _, retired := range []bool{false, true} {
		fake := newFakeDex()
		fake.chapters = `{"id":"r","type":"chapter","attributes":{"chapter":"1","pages":3,"isUnavailable":true}},` +
			`{"id":"e","type":"chapter","attributes":{"chapter":"2","pages":0,"externalUrl":"https://x.test"}},` +
			`{"id":"a","type":"chapter","attributes":{"chapter":"3","pages":3}}`
		options := mango.DefaultOptions()
		options.HTTPClient = &http.Client{Transport: fake}
		options.Filter.ShowUnavailableChapters = false
		options.Filter.ShowRetiredChapters = retired
		fns := load(t, Loader(options).(*mango.Loader), nil)
		manga := mango.Manga{ID: "m"}
		fns.MangaVolumes(context.Background(), mango.Store{}, manga)
		chapters, err := fns.VolumeChapters(context.Background(), mango.Store{}, mango.Volume{Number: 1, Manga_: &manga})
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{"/manga/m/aggregate", "/chapter"} {
			query, _ := url.ParseQuery(fake.queries[path])
			if got := query.Get("includeUnavailable") == "1"; got != retired {
				t.Errorf("retired %t, %s: got %s", retired, path, fake.queries[path])
			}
		}
		var got []string
		for _, c := range chapters {
			got = append(got, c.(*mango.Chapter).ID+":"+c.Info().Unavailable)
		}
		// the fake answers retired chapters either way, the server only does when asked
		if want := "r:retired,a:"; strings.Join(got, ",") != want {
			t.Errorf("retired %t: wanted %s, got %v", retired, want, got)
		}
	}
}

func TestMangaByIDWithoutManga(t *testing.T) {
	fns := loadFake(t, newFakeDex(), nil)
	if manga, found, err := fns.MangaByID(context.Background(), mango.Store{}, "empty"); err == nil || !strings.Contains(err.Error(), "no manga in the response") || found {
		t.Errorf("got %v, %v, %v, want an error", manga, found, err)
	}
}
