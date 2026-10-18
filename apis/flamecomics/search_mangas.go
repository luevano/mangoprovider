package flamecomics

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
)

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

// SearchMangas filters the whole catalog (/browse lists every series).
func (f *flame) SearchMangas(ctx context.Context, store mango.Store, query string) ([]mangadata.Manga, error) {
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

// MangaByID gets the series by its number.
func (f *flame) MangaByID(ctx context.Context, _ mango.Store, id string) (mangadata.Manga, bool, error) {
	var props struct {
		Series series `json:"series"`
	}
	if err := f.page(ctx, "series/"+url.PathEscape(id), &props); err != nil {
		if errors.Is(err, mango.ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	// next.js may answer an unknown series with an empty page
	if props.Series.ID == 0 {
		return nil, false, nil
	}
	return toManga(props.Series), true, nil
}

func toManga(s series) *mango.Manga {
	id := strconv.Itoa(s.ID)
	cover := ""
	if s.Cover != "" {
		cover = cdn + id + "/" + s.Cover
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
		PublicationStatus: mango.ParseStatus(s.Status),
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
