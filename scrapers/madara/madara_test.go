package madara

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper"
)

func selection(t *testing.T, html string) *goquery.Selection {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	return doc.Selection
}

func TestImageURL(t *testing.T) {
	for html, want := range map[string]string{
		`<img data-src=" https://a/1.jpg " src="data:x">`:                                 "https://a/1.jpg",
		`<img data-lazy-src="https://a/2.jpg">`:                                           "https://a/2.jpg",
		`<img srcset="https://a/s.jpg 175w, https://a/l.jpg 350w" src="https://a/s.jpg">`: "https://a/l.jpg",
		`<img srcset="https://a/1x.jpg 1x, https://a/2x.jpg 2x">`:                         "https://a/2x.jpg",
		`<img src="https://a/3.jpg">`:                                                     "https://a/3.jpg",
		`<p></p>`:                                                                         "",
	} {
		if got := ImageURL(selection(t, html).Find("img")); got != want {
			t.Errorf("%s: got %q, want %q", html, got, want)
		}
	}
}

func TestOwnText(t *testing.T) {
	if got := OwnText(selection(t, `<h1> Solo  Leveling <span>HOT</span></h1>`).Find("h1")); got != "Solo Leveling" {
		t.Errorf("got %q", got)
	}
}

func TestPostID(t *testing.T) {
	for html, want := range map[string]string{
		`<div id="manga-chapters-holder" data-id="12"></div>`:                 "12",
		`<input class="rating-post-id" value="13">`:                           "13",
		`<a data-post="14"></a>`:                                              "14",
		`<link rel="shortlink" href="https://site/?p=15">`:                    "15",
		`<div id="manga-chapters-holder" data-id=""></div><a data-post="16">`: "16",
		`<p></p>`: "",
	} {
		if got := PostID(selection(t, html)); got != want {
			t.Errorf("%s: got %q, want %q", html, got, want)
		}
	}
}

func TestChapterDate(t *testing.T) {
	now := time.Date(2025, 10, 13, 12, 0, 0, 0, time.UTC)
	o := Options{}.withDefaults()
	for html, want := range map[string]string{
		`<li><span class="chapter-release-date"><i>October 9, 2025</i></span></li>`:                     "2025-10-09",
		`<li><span class="chapter-release-date"><a title="3 hours ago"><img alt="new"></a></span></li>`: "2025-10-13",
		`<li><img class="thumb" alt="x"><img alt="5 days ago"></li>`:                                    "2025-10-08",
		`<li><span class="chapter-release-date">soon</span></li>`:                                       "",
	} {
		if got := o.chapterDate(selection(t, html).Find("li"), now).String(); got != want {
			t.Errorf("%s: got %q, want %q", html, got, want)
		}
	}
}

// madara is a fake Madara site: /manga/<slug>/ pages, the ajax endpoints, an html search.
type madara struct {
	mu       sync.Mutex
	requests []string
}

func (m *madara) serve(t *testing.T) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" && r.URL.RawQuery == "" {
			return
		}
		r.ParseForm()
		m.mu.Lock()
		m.requests = append(m.requests, r.Method+" "+r.URL.RequestURI()+" "+r.PostForm.Encode())
		m.mu.Unlock()
		chapters := `<ul><li class="wp-manga-chapter"><a href="/manga/a-b/chapter-2/">Chapter 2</a><span class="chapter-release-date"><i>March 3, 2024</i></span></li>` +
			`<li class="wp-manga-chapter"><a href="/manga/a-b/chapter-1/">Chapter 1</a></li></ul>`
		switch {
		case r.URL.Path == "/manga/a-b/":
			w.Write([]byte(`<div class="post-title"><h1>A B</h1></div><input class="rating-post-id" value="77">` + chapters))
		case r.URL.Path == "/old/a-b/":
			http.Redirect(w, r, "/manga/a-b/", http.StatusMovedPermanently)
		case r.URL.Path == "/manga/empty/":
			w.Write([]byte(`<p>not a manga</p>`))
		case r.URL.Path == "/manga/a-b/ajax/chapters/" && r.Method == http.MethodPost:
			w.Write([]byte(chapters))
		case r.URL.Path == "/wp-admin/admin-ajax.php" && r.PostForm.Get("action") == "manga_get_chapters" && r.PostForm.Get("manga") == "77":
			w.Write([]byte(chapters))
		case r.URL.Path == "/index.php" && r.PostForm.Get("manga-core") == "a-b":
			w.Write([]byte(chapters))
		case r.URL.Path == "/" || r.URL.Path == "/page/2/":
			// MadaraNoAjax's search: search cards, a next link on the first page
			html := `<div class="c-tabs-item__content"><div class="post-title"><a href="/manga/a-b/">A B</a></div><img data-src="https://cdn/a.jpg"></div>`
			if r.URL.Path == "/" {
				html += `<div class="nav-previous"><a href="/page/2/?s=a+b&post_type=wp-manga">Older</a></div>`
			} else {
				html = strings.ReplaceAll(html, "a-b", "c-d")
			}
			w.Write([]byte(html))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestChapterModes(t *testing.T) {
	for mode, want := range map[ChapterMode]string{
		MangaPage:      "GET /manga/a-b/ ",
		MangaAjax:      "POST /manga/a-b/ajax/chapters/ ",
		AdminAjax:      "GET /manga/a-b/ ,POST /wp-admin/admin-ajax.php action=manga_get_chapters&manga=77",
		MangaAjaxQuery: "POST /index.php maction=get_chapters&manga-core=a-b&manga_ajax=1",
	} {
		fake := &madara{}
		server := fake.serve(t)
		s, err := scraper.NewScraper(context.Background(), Config(Options{ID: "test", BaseURL: server.URL + "/", ChapterMode: mode}), mango.DefaultOptions(), nil)
		if err != nil {
			t.Fatal(err)
		}
		manga := mango.Manga{ID: "a-b", URL: server.URL + "/manga/a-b/"}
		chapters, err := s.VolumeChapters(context.Background(), mango.Store{}, mango.Volume{Number: 1, Manga_: &manga})
		if err != nil || len(chapters) != 2 {
			t.Errorf("mode %d: got %v, %v", mode, chapters, err)
			continue
		}
		if c := chapters[1].(*mango.Chapter); c.ID != "chapter-2" || c.Number != 2 || c.Date.String() != "2024-03-03" {
			t.Errorf("mode %d: got %+v", mode, c)
		}
		if got := strings.Join(fake.requests, ","); got != want {
			t.Errorf("mode %d: requests %q, want %q", mode, got, want)
		}
	}
}

func TestNoAjaxSearch(t *testing.T) {
	fake := &madara{}
	server := fake.serve(t)
	s, err := scraper.NewScraper(context.Background(), Config(Options{ID: "test", BaseURL: server.URL + "/", NoAjax: true}), mango.DefaultOptions(), nil)
	if err != nil {
		t.Fatal(err)
	}
	mangas, err := s.SearchMangas(context.Background(), mango.Store{}, "a b")
	if err != nil || len(mangas) != 2 {
		t.Fatalf("got %v, %v", mangas, err)
	}
	if info := mangas[0].Info(); info.ID != "a-b" || info.Title != "A B" || info.Cover != "https://cdn/a.jpg" {
		t.Errorf("got %+v", info)
	}
	want := "GET /?post_type=wp-manga&s=a+b ,GET /page/2/?post_type=wp-manga&s=a+b "
	if got := strings.Join(fake.requests, ","); got != want {
		t.Errorf("requests %q, want %q", got, want)
	}
}

func TestAjaxSearchForm(t *testing.T) {
	o := Options{BaseURL: "https://site/"}.withDefaults()
	r := o.ajaxSearch(o.BaseURL, "a b", 2)
	if r.URL != "https://site/wp-admin/admin-ajax.php" || r.Form.Get("page") != "1" || r.Form.Get("vars[s]") != "a b" ||
		r.Form.Get("vars[posts_per_page]") != "25" || r.Form.Get("vars[meta_query][0][value]") != "manga" || r.Headers["X-Requested-With"] != "XMLHttpRequest" {
		t.Errorf("got %+v", r)
	}
	o.KeepNonManga = true
	if r := o.ajaxSearch(o.BaseURL, "a", 1); r.Form.Has("vars[meta_query][0][key]") {
		t.Errorf("non manga items filtered: %v", r.Form)
	}
}

// a page without the title isn't a manga
func TestMangaByIDNoTitle(t *testing.T) {
	fake := &madara{}
	server := fake.serve(t)
	s, err := scraper.NewScraper(context.Background(), Config(Options{ID: "test", BaseURL: server.URL + "/"}), mango.DefaultOptions(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if manga, found, err := s.MangaByID(context.Background(), mango.Store{}, "a-b"); err != nil || !found || manga.Info().Title != "A B" {
		t.Errorf("got %v, %v, %v", manga, found, err)
	}
	if _, _, err := s.MangaByID(context.Background(), mango.Store{}, "empty"); err == nil || !strings.Contains(err.Error(), "no manga in the page") {
		t.Errorf("got %v", err)
	}
}

// a manga that moved: its chapters are asked where it is now (a POST isn't redirected)
func TestMovedManga(t *testing.T) {
	fake := &madara{}
	server := fake.serve(t)
	s, err := scraper.NewScraper(context.Background(), Config(Options{ID: "test", BaseURL: server.URL + "/", ChapterMode: MangaAjax}), mango.DefaultOptions(), nil)
	if err != nil {
		t.Fatal(err)
	}
	volumes, err := s.MangaVolumes(context.Background(), mango.Store{}, mango.Manga{ID: "a-b", URL: server.URL + "/old/a-b/"})
	if err != nil || len(volumes) != 1 {
		t.Fatalf("got %v, %v", volumes, err)
	}
	chapters, err := s.VolumeChapters(context.Background(), mango.Store{}, *volumes[0].(*mango.Volume))
	if err != nil || len(chapters) != 2 {
		t.Errorf("got %v, %v (requests %q)", chapters, err, fake.requests)
	}
}
