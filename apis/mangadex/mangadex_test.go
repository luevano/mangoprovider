package mangadex

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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
}

func newFakeDex() *fakeDex {
	return &fakeDex{requests: map[string]int{}, nodeStatus: http.StatusOK, total: 10000}
}

func (f *fakeDex) count(path string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests[path]
}

func (f *fakeDex) RoundTrip(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	f.requests[r.URL.Path]++
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
