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

	mangas, err = s.search(ctx, searchURL)
	if err != nil {
		return nil, err
	}

	// could be a markup change on the site, try again next time
	if len(mangas) == 0 {
		return mangas, nil
	}
	return mangas, store.SetMangas(searchURL, mangas)
}

// search: The mangas in the search's pages, see Pagination.
func (s *Scraper) search(ctx context.Context, searchURL string) ([]mangadata.Manga, error) {
	pagination := s.config.SearchPagination
	maxPages := 1
	if pagination != nil && (pagination.NextSelector != "" || pagination.URL != nil) {
		maxPages = pagination.MaxPages
		if maxPages <= 0 {
			maxPages = defaultMaxPages
		}
	}

	var mangas []mangadata.Manga
	seen := map[string]bool{}
	pageURL := searchURL
	for n := 1; ; n++ {
		if n > 1 {
			s.log.Progress(ctx, n, maxPages, "%d results so far, search page", len(mangas))
		}
		var hasNext bool
		var nextURL string
		err := s.scrape(s.mangaAction(ctx), pageURL, s.config.BaseURL, func(p *page) error {
			added := 0
			for _, manga := range s.parseMangas(p) {
				if id := manga.Info().ID; !seen[id] {
					seen[id] = true
					mangas = append(mangas, manga)
					added++
				}
			}
			// a page without new mangas is the last, even if it links to another
			hasNext = added > 0
			if n >= maxPages || !hasNext || pagination.NextSelector == "" {
				return nil
			}
			link := p.doc.Find(pagination.NextSelector).First()
			hasNext = link.Length() > 0
			href, ok := link.Attr("href")
			if !ok {
				href = link.Find("a[href]").First().AttrOr("href", "")
			}
			nextURL = p.absoluteURL(href)
			return nil
		})
		// a page past the last one can be a 404
		if n > 1 && errors.Is(err, mango.ErrNotFound) {
			break
		}
		if err != nil {
			return nil, err
		}
		if n >= maxPages || !hasNext {
			break
		}
		if pagination.URL != nil {
			if pageURL, err = pagination.URL(searchURL, n+1); err != nil {
				return nil, err
			}
		} else if nextURL != "" {
			pageURL = nextURL
		} else {
			s.log.Log("search page %d: %q has no link to the next page", n, pagination.NextSelector)
			break
		}
	}
	return mangas, nil
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
