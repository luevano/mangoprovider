package mangathemesia

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/scrapertest"
)

func selection(t *testing.T, html string) *goquery.Selection {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	return doc.Selection
}

var files = map[string]string{
	"/manga/?title=solo&page=1": "search.html",
	"/manga/?title=solo&page=2": "search-2.html",
	"/?s=solo":                  "search.html",
	"/page/2/?s=solo":           "search-2.html",
	"/manga/solo-leveling/":     "manga.html",
	"/solo-leveling-chapter-3/": "chapter.html",
	"/solo-leveling-chapter-2/": "chapter-script.html",
	"/solo-leveling-prologue/":  "chapter-base64.html",
}

func TestSearch(t *testing.T) {
	for _, titleFilter := range []bool{false, true} {
		testSearch(t, titleFilter)
	}
}

func testSearch(t *testing.T, titleFilter bool) {
	s, _ := scrapertest.New(t, Config(Options{ID: "test", TitleFilter: titleFilter}), files)
	mangas, err := s.SearchMangas(context.Background(), mango.Store{}, "solo")
	if err != nil || len(mangas) != 3 {
		t.Fatalf("title filter %v: got %v, %v", titleFilter, mangas, err)
	}
	for i, want := range []mangadata.MangaInfo{
		{ID: "solo-leveling", Title: "Solo Leveling", Cover: "https://cdn.example/solo.webp"},
		{ID: "the-greatest-estate-developer", Title: "The Greatest Estate Developer", Cover: "https://cdn.example/ged.jpg"},
		{ID: "solo-leveling-ragnarok", Title: "Solo Leveling: Ragnarok", Cover: "https://cdn.example/rag.webp"},
	} {
		got := mangas[i].Info()
		if got.ID != want.ID || got.Title != want.Title || got.Cover != want.Cover {
			t.Errorf("title filter %v, manga %d: got %+v, want %+v", titleFilter, i, got, want)
		}
	}
}

func TestMangaChaptersPages(t *testing.T) {
	info := libmangal.ProviderInfo{ID: "test", Name: "Test Scans"}
	s, base := scrapertest.New(t, Mangathemesia(info, "/manga/"), files)
	ctx := context.Background()

	manga, found, err := s.MangaByID(ctx, mango.Store{}, "solo-leveling")
	if err != nil || !found {
		t.Fatalf("got %v, %v, %v", manga, found, err)
	}
	if got := manga.Info(); got.Title != "Solo Leveling" || got.Cover != "https://cdn.example/solo-big.webp" {
		t.Errorf("got %+v", got)
	}
	meta := manga.Metadata().(*mangadata.Metadata)
	if meta.PublicationStatus != metadata.StatusFinished || meta.PublicationFormat != "Manhwa" || meta.CountryOfOrigin != "KR" ||
		strings.Join(meta.AuthorList, ",") != "Chugong" || len(meta.ArtistList) != 0 ||
		strings.Join(meta.GenreList, ",") != "Action,Fantasy" || strings.Join(meta.Synonyms, ",") != "나 혼자만 레벨업,Only I Level Up" ||
		meta.Summary != "Ten years ago, the Gate appeared.\n\nE-rank hunter Sung Jinwoo." {
		t.Errorf("got %#v", *meta)
	}

	m := manga.(*mango.Manga)
	volumes, err := s.MangaVolumes(ctx, mango.Store{}, *m)
	if err != nil || len(volumes) != 1 {
		t.Fatalf("got %v, %v", volumes, err)
	}
	chapters, err := s.VolumeChapters(ctx, mango.Store{}, *volumes[0].(*mango.Volume))
	// the locked chapter has no link
	if err != nil || len(chapters) != 3 {
		t.Fatalf("got %v, %v", chapters, err)
	}
	for i, want := range []struct {
		id, date string
		number   float32
	}{
		{"solo-leveling-chapter-2", "2024-03-03", 2},
		{"solo-leveling-chapter-3", "2025-10-09", 3},
	} {
		c := chapters[i+1].(*mango.Chapter)
		if c.ID != want.id || c.Number != want.number || c.Date.String() != want.date || c.ScanlationGroup != "Test Scans" {
			t.Errorf("chapter %d: got %+v", i, c)
		}
	}
	if meta.DateStart != (metadata.Date{Year: 2018}) {
		t.Errorf("start: got %v", meta.DateStart)
	}

	for i, want := range [][]string{
		{"https://cdn.example/p/01.jpg", "https://cdn.example/p/02.jpg"},
		{"https://cdn.example/ch2/01.jpg", base + "wp-content/uploads/ch2/02.png"},
		{"https://cdn.example/ch3/01.jpg", "https://cdn.example/ch3/02.webp"},
	} {
		pages, err := s.ChapterPages(ctx, mango.Store{}, *chapters[i].(*mango.Chapter))
		if err != nil {
			t.Fatalf("chapter %d: %v", i, err)
		}
		var got []string
		for _, p := range pages {
			got = append(got, p.(*mango.Page).URL)
		}
		if strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("chapter %d: got %v, want %v", i, got, want)
		}
	}
}

func TestChapterDate(t *testing.T) {
	now := time.Date(2025, 10, 13, 12, 0, 0, 0, time.UTC)
	o := Options{}.withDefaults()
	doc := selection(t, `<div class="listinfo"><time itemprop="dateModified" datetime="2025-10-01T08:00:00+00:00"></time></div>`+
		`<div id="chapterlist"><ul><li id="a"><span class="chapterdate"></span></li><li id="b"><span class="chapterdate">2 days ago</span></li><li id="c"></li></ul></div>`)
	for id, want := range map[string]string{"a": "2025-10-01", "b": "2025-10-11", "c": ""} {
		if got := o.chapterDate(doc.Find("#"+id), now).String(); got != want {
			t.Errorf("%s: got %q, want %q", id, got, want)
		}
	}
}

func TestIDs(t *testing.T) {
	o := Options{MangaDir: "series"}.withDefaults()
	for _, tt := range []struct{ url, manga, chapter string }{
		{"https://site/series/a-b/", "a-b", "a-b"},
		{"https://site/a-b-chapter-1/", "a-b-chapter-1", "a-b-chapter-1"},
		{"https://site/", "", ""},
	} {
		if got := o.mangaID(tt.url); got != tt.manga {
			t.Errorf("manga id of %q: got %q, want %q", tt.url, got, tt.manga)
		}
	}
	if got := o.chapterID("https://site/series/a-b/"); got != "" {
		t.Errorf("the manga page as a chapter: got %q", got)
	}
	if got := o.chapterID("https://site/a-b-chapter-1/"); got != "a-b-chapter-1" {
		t.Errorf("got %q", got)
	}
}

func TestStatus(t *testing.T) {
	for text, want := range map[string]metadata.Status{
		"Ongoing": metadata.StatusReleasing, " Completed ": metadata.StatusFinished, "Dropped": metadata.StatusCancelled,
		"Hiatus": metadata.StatusHiatus, "Season End": metadata.StatusHiatus, "soon": metadata.Status("UNKNOWN"),
	} {
		if got := Status(text); got != want {
			t.Errorf("%q: got %q, want %q", text, got, want)
		}
	}
}
