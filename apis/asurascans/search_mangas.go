package asurascans

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
)

type series struct {
	Slug      string   `json:"slug"`
	Title     string   `json:"title"`
	AltTitles []string `json:"alt_titles"`
	// the series endpoint joins them instead
	AlternativeTitles string `json:"alternative_titles"`
	Description       string `json:"description"`
	Cover             string `json:"cover"`
	Status            string `json:"status"`
	Type              string `json:"type"`
	Author            string `json:"author"`
	Artist            string `json:"artist"`
	PublicURL         string `json:"public_url"`
	Genres            []struct {
		Name string `json:"name"`
	} `json:"genres"`
}

// SearchMangas uses the site's search.
func (a *asura) SearchMangas(ctx context.Context, store mango.Store, query string) ([]mangadata.Manga, error) {
	var mangas []mangadata.Manga
	if found, err := store.GetMangas(query, query, &mangas); err != nil || found {
		return mangas, err
	}
	mangas, err := a.search(ctx, query)
	if err != nil {
		return nil, err
	}
	return mangas, store.SetMangas(query, mangas)
}

// MangaByID gets the series by its slug (old asuracomic.net ids with their "-<hash>" suffix
// redirect to the series too).
func (a *asura) MangaByID(ctx context.Context, _ mango.Store, id string) (mangadata.Manga, bool, error) {
	var resp struct {
		Series series `json:"series"`
	}
	if err := a.get(ctx, "series/"+url.PathEscape(id), &resp); err != nil {
		if errors.Is(err, mango.ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	// seen once: a 200 without the series, which would be an empty manga (empty cache keys)
	if resp.Series.Slug == "" {
		return nil, false, fmt.Errorf("series %q: no series in the response", id)
	}
	return toManga(resp.Series), true, nil
}

func (a *asura) search(ctx context.Context, query string) ([]mangadata.Manga, error) {
	var mangas []mangadata.Manga
	for page := 1; page <= searchPages; page++ {
		if page > 1 {
			a.log.Progress(ctx, page, searchPages, "%d results so far, search page", len(mangas))
		}
		var resp struct {
			Data []series `json:"data"`
			Meta struct {
				HasMore bool `json:"has_more"`
			} `json:"meta"`
		}
		params := url.Values{"q": {query}, "page": {strconv.Itoa(page)}}
		if err := a.get(ctx, "search?"+params.Encode(), &resp); err != nil {
			return nil, err
		}
		// novels have their own endpoint, search only has comics
		for _, s := range resp.Data {
			mangas = append(mangas, toManga(s))
		}
		if !resp.Meta.HasMore {
			break
		}
	}
	return mangas, nil
}

func toManga(s series) *mango.Manga {
	var alt, genres, authors, artists []string
	if len(s.AltTitles) == 0 && s.AlternativeTitles != "" {
		s.AltTitles = strings.Split(s.AlternativeTitles, " • ")
	}
	for _, t := range s.AltTitles {
		if t = strings.TrimSpace(t); t != "" {
			alt = append(alt, t)
		}
	}
	for _, g := range s.Genres {
		genres = append(genres, g.Name)
	}
	if s.Author != "" {
		authors = []string{s.Author}
	}
	if s.Artist != "" {
		artists = []string{s.Artist}
	}
	// the public url has a "-<hash>" suffix the api doesn't need
	mangaURL := website + "comics/" + s.Slug
	if s.PublicURL != "" {
		mangaURL = strings.TrimSuffix(website, "/") + s.PublicURL
	}
	var meta metadata.Metadata = &mangadata.Metadata{
		EnglishTitle:      s.Title,
		Synonyms:          alt,
		PublicationFormat: s.Type,
		Summary:           s.Description,
		CoverImage:        s.Cover,
		AuthorList:        authors,
		ArtistList:        artists,
		GenreList:         genres,
		PublicationStatus: mango.ParseStatus(s.Status),
		CountryOfOrigin:   mango.CountryOfType(s.Type),
		SourceURL:         mangaURL,
		ProviderID:        s.Slug,
		ProviderIDCode:    "asura",
	}
	return &mango.Manga{
		Title:     s.Title,
		URL:       mangaURL,
		ID:        s.Slug,
		Cover:     s.Cover,
		Metadata_: &meta,
	}
}
