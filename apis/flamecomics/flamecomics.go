// Package flamecomics reads Flame Comics' pages, a Next.js site: every page embeds its
// data as JSON (__NEXT_DATA__), so there's nothing to scrape from the html.
package flamecomics

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
)

const (
	website = "https://flamecomics.xyz/"
	cdn     = "https://cdn.flamecomics.xyz/uploads/images/series/"
)

var providerInfo = libmangal.ProviderInfo{
	ID:          mango.BundleID + "-flamecomics",
	Name:        "FlameComics",
	Version:     "0.4.0",
	Description: "FlameComics, from the data its pages embed",
	Website:     website,
}

var nextData = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__" type="application/json">(.*?)</script>`)

type flame struct {
	client    *http.Client
	userAgent string
	// base is the website, a test server in tests
	base string
}

func Loader(options mango.Options) libmangal.ProviderLoader {
	f := flame{client: options.HTTPClient, userAgent: options.UserAgent, base: website}
	return &mango.Loader{
		ProviderInfo: providerInfo,
		Options:      options,
		F: func(context.Context, *mango.Logger) (mango.Functions, error) {
			return mango.Functions{
				SearchMangas:   f.SearchMangas,
				MangaVolumes:   f.MangaVolumes,
				VolumeChapters: f.VolumeChapters,
				ChapterPages:   f.ChapterPages,
			}, nil
		},
	}
}

type series struct {
	ID          int      `json:"series_id"`
	Title       string   `json:"title"`
	AltTitles   []string `json:"altTitles"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Categories  []string `json:"categories"`
	Country     string   `json:"country"`
	Author      []string `json:"author"`
	Artist      []string `json:"artist"`
	Year        int      `json:"year"`
	Status      string   `json:"status"`
	Cover       string   `json:"cover"`
	LastEdit    int64    `json:"last_edit"`
}

type chapter struct {
	Number  string `json:"chapter"`
	Title   string `json:"title"`
	Token   string `json:"token"`
	Release int64  `json:"release_date"`
}

type chapterPage struct {
	Token  string `json:"token"`
	Images map[string]struct {
		Name string `json:"name"`
	} `json:"images"`
}

// page gets a page of the site and decodes the props it embeds into v.
func (f *flame) page(ctx context.Context, path string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", f.userAgent)
	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("GET /%s: %w", path, mango.ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET /%s: %d %s", path, resp.StatusCode, strings.ToLower(http.StatusText(resp.StatusCode)))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	m := nextData.FindSubmatch(body)
	if m == nil {
		return fmt.Errorf("GET /%s: no page data", path)
	}
	var data struct {
		Props struct {
			PageProps json.RawMessage `json:"pageProps"`
		} `json:"props"`
	}
	if err := json.Unmarshal(m[1], &data); err != nil {
		return fmt.Errorf("GET /%s: %w", path, err)
	}
	if err := json.Unmarshal(data.Props.PageProps, v); err != nil {
		return fmt.Errorf("GET /%s: %w", path, err)
	}
	return nil
}

// SearchMangas filters the whole catalog (/browse lists every series), or gets one by "mid:".
func (f *flame) SearchMangas(ctx context.Context, store mango.Store, query string) ([]mangadata.Manga, error) {
	if id, byID := mango.ReNamedGroups(mango.MangaQueryIDRegex, query)[mango.MangaQueryID]; byID {
		var props struct {
			Series series `json:"series"`
		}
		if err := f.page(ctx, "series/"+id, &props); err != nil {
			return nil, err
		}
		return []mangadata.Manga{toManga(props.Series)}, nil
	}

	var mangas []mangadata.Manga
	if found, err := store.GetMangas(query, query, &mangas); err != nil || found {
		return mangas, err
	}
	var props struct {
		Series []series `json:"series"`
	}
	if err := f.page(ctx, "browse", &props); err != nil {
		return nil, err
	}
	q := strings.ToLower(strings.TrimSpace(query))
	for _, s := range props.Series {
		// novels have no images
		if strings.Contains(s.Type, "Novel") {
			continue
		}
		titles := append([]string{s.Title}, s.AltTitles...)
		if slices.ContainsFunc(titles, func(t string) bool { return strings.Contains(strings.ToLower(t), q) }) {
			mangas = append(mangas, toManga(s))
		}
	}
	return mangas, store.SetMangas(query, mangas)
}

func (f *flame) MangaVolumes(_ context.Context, _ mango.Store, manga mango.Manga) ([]mangadata.Volume, error) {
	// no volumes
	return []mangadata.Volume{&mango.Volume{Number: 1, None: true, Manga_: &manga}}, nil
}

func (f *flame) VolumeChapters(ctx context.Context, store mango.Store, volume mango.Volume) ([]mangadata.Chapter, error) {
	var chapters []mangadata.Chapter
	id := volume.Manga_.ID
	if found, err := store.GetChapters(id, volume, &chapters); err != nil || found {
		return chapters, err
	}
	var props struct {
		Chapters []chapter `json:"chapters"`
	}
	if err := f.page(ctx, "series/"+id, &props); err != nil {
		return nil, err
	}
	for _, c := range props.Chapters {
		number, err := strconv.ParseFloat(c.Number, 32)
		if err != nil {
			continue
		}
		date := time.Unix(c.Release, 0)
		chapters = append(chapters, &mango.Chapter{
			Title:           c.Title,
			ID:              c.Token,
			URL:             fmt.Sprintf("%sseries/%s/%s", website, id, c.Token),
			Number:          float32(number),
			Date:            metadata.Date{Year: date.Year(), Month: int(date.Month()), Day: date.Day()},
			ScanlationGroup: "Flame Comics",
			Volume_:         &volume,
		})
	}
	// newest first on the site
	slices.Reverse(chapters)
	return chapters, store.SetChapters(id, chapters)
}

func (f *flame) ChapterPages(ctx context.Context, _ mango.Store, chapter mango.Chapter) ([]mangadata.Page, error) {
	id := chapter.Volume_.Manga_.ID
	var props struct {
		Chapter chapterPage `json:"chapter"`
	}
	if err := f.page(ctx, fmt.Sprintf("series/%s/%s", id, chapter.ID), &props); err != nil {
		return nil, err
	}
	// the images are an object keyed by their position
	keys := make([]int, 0, len(props.Chapter.Images))
	for k := range props.Chapter.Images {
		if n, err := strconv.Atoi(k); err == nil {
			keys = append(keys, n)
		}
	}
	slices.Sort(keys)
	var pages []mangadata.Page
	for _, k := range keys {
		url := fmt.Sprintf("%s%s/%s/%s", cdn, id, chapter.ID, props.Chapter.Images[strconv.Itoa(k)].Name)
		ext, err := mango.ImageExtension(url)
		if err != nil {
			return nil, err
		}
		pages = append(pages, &mango.Page{
			Ext:      ext,
			URL:      url,
			Headers:  map[string]string{"Referer": website, "User-Agent": f.userAgent},
			Chapter_: &chapter,
		})
	}
	return pages, nil
}

func toManga(s series) *mango.Manga {
	id := strconv.Itoa(s.ID)
	cover := ""
	if s.Cover != "" {
		cover = cdn + id + "/" + s.Cover
	}
	status := metadata.Status("UNKNOWN")
	switch strings.ToLower(s.Status) {
	case "ongoing":
		status = metadata.StatusReleasing
	case "completed":
		status = metadata.StatusFinished
	case "hiatus":
		status = metadata.StatusHiatus
	case "dropped", "cancelled":
		status = metadata.StatusCancelled
	case "coming soon":
		status = metadata.StatusNotYetReleased
	}
	var alt []string
	for _, t := range s.AltTitles {
		if t = strings.TrimSpace(t); t != "" {
			alt = append(alt, t)
		}
	}
	var meta metadata.Metadata = &mangadata.Metadata{
		EnglishTitle:      s.Title,
		Synonyms:          alt,
		PublicationFormat: s.Type,
		Summary:           s.Description,
		CoverImage:        cover,
		AuthorList:        s.Author,
		ArtistList:        s.Artist,
		GenreList:         s.Categories,
		DateStart:         metadata.Date{Year: s.Year},
		PublicationStatus: status,
		CountryOfOrigin:   s.Country,
		SourceURL:         website + "series/" + id,
		ProviderID:        id,
		ProviderIDCode:    "fc",
	}
	return &mango.Manga{
		Title:     s.Title,
		URL:       website + "series/" + id,
		ID:        id,
		Cover:     cover,
		Metadata_: &meta,
	}
}
