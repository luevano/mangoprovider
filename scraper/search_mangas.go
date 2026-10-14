package scraper

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/headless/rod"
)

func (s *Scraper) SearchMangas(ctx context.Context, store mango.Store, query string) ([]mangadata.Manga, error) {
	if id, byID := mango.ReNamedGroups(mango.MangaQueryIDRegex, query)[mango.MangaQueryID]; byID {
		return mango.SearchByID(s.MangaByID(ctx, store, id))
	}

	var mangas []mangadata.Manga
	searchURL, err := s.config.GenerateSearchURL(s.config.BaseURL, query)
	if err != nil {
		return nil, err
	}
	found, err := store.GetMangas(searchURL, query, &mangas)
	if err != nil {
		return nil, err
	}
	if found {
		return mangas, nil
	}

	err = s.scrape(s.mangaAction(ctx), searchURL, s.config.BaseURL, func(p *page) error {
		mangas = s.parseMangas(p)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// could be a markup change on the site, try again next time
	if len(mangas) == 0 {
		return mangas, nil
	}
	return mangas, store.SetMangas(searchURL, mangas)
}

// MangaByID scrapes the manga's page, found false if the site answers 404.
func (s *Scraper) MangaByID(ctx context.Context, store mango.Store, id string) (mangadata.Manga, bool, error) {
	id = strings.Trim(id, "/")
	if s.config.GenerateSearchByIDURL == nil {
		return nil, false, fmt.Errorf("search by id: %w (no GenerateSearchByIDURL)", mango.ErrNotSupported)
	}
	if s.config.MangaByIDExtractor == nil {
		return nil, false, fmt.Errorf("search by id: %w (no MangaByIDExtractor)", mango.ErrNotSupported)
	}
	mangaURL, err := s.config.GenerateSearchByIDURL(s.config.BaseURL, mango.EscapePath(id))
	if err != nil {
		return nil, false, err
	}

	cacheID := "mid:" + id
	var mangas []mangadata.Manga
	if found, err := store.GetMangas(cacheID, cacheID, &mangas); err != nil {
		return nil, false, err
	} else if found && len(mangas) == 1 {
		return mangas[0], true, nil
	}

	var manga *mango.Manga
	notFound := false
	err = s.scrape(s.mangaAction(ctx), mangaURL, s.config.BaseURL, func(p *page) error {
		// some sites (weebcentral) redirect unknown mangas to an error page that answers 200
		if errorPagePath.MatchString(p.url.Path) {
			notFound = true
			return nil
		}
		manga = s.parseManga(p, id)
		return nil
	})
	if errors.Is(err, mango.ErrNotFound) || (err == nil && notFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if manga == nil {
		return nil, false, fmt.Errorf("GET %s: no manga in the page", mangaURL)
	}
	return manga, true, store.SetMangas(cacheID, []mangadata.Manga{manga})
}

var errorPagePath = regexp.MustCompile(`^/(400|404)/?$`)

// TODO: define metadata scraping logic
// parseManga: The manga in its page, nil if it isn't there.
func (s *Scraper) parseManga(p *page, id string) *mango.Manga {
	selection := p.doc.Find(s.config.MangaByIDExtractor.Selector).First()
	if selection.Length() == 0 {
		return nil
	}
	var meta metadata.Metadata = &mangadata.Metadata{}
	return &mango.Manga{
		Title:     mango.CleanString(s.config.MangaByIDExtractor.Title(selection)),
		URL:       p.url.String(),
		ID:        id,
		Cover:     s.config.MangaByIDExtractor.Cover(selection),
		Metadata_: &meta,
	}
}

// TODO: define metadata scraping logic
// parseMangas: The mangas in a search page.
func (s *Scraper) parseMangas(p *page) []mangadata.Manga {
	var mangas []mangadata.Manga
	p.doc.Find(s.config.MangaExtractor.Selector).Each(func(_ int, selection *goquery.Selection) {
		url := p.absoluteURL(s.config.MangaExtractor.URL(selection))
		id := s.config.MangaExtractor.ID(url)
		if id == "" {
			s.log.Log("skipping manga %q: no id in %q", mango.CleanString(selection.Text()), url)
			return
		}
		var meta metadata.Metadata = &mangadata.Metadata{}
		mangas = append(mangas, &mango.Manga{
			Title:     mango.CleanString(s.config.MangaExtractor.Title(selection)),
			URL:       url,
			ID:        id,
			Cover:     s.config.MangaExtractor.Cover(selection),
			Metadata_: &meta,
		})
	})
	return mangas
}

// mangaAction: ctx with the MangaExtractor's Action, if any.
func (s *Scraper) mangaAction(ctx context.Context) context.Context {
	if s.config.MangaExtractor == nil {
		return ctx
	}
	return rod.WithAction(ctx, s.config.MangaExtractor.Action)
}
