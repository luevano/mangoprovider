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
	first, next, err := s.searchRequests(query)
	if err != nil {
		return nil, err
	}
	found, err := store.GetMangas(first.key(), query, &mangas)
	if err != nil {
		return nil, err
	}
	if found {
		s.withCachedMetadata(store, mangas)
		return mangas, nil
	}

	mangas, err = s.search(ctx, first, next)
	if err != nil {
		return nil, err
	}
	s.withCachedMetadata(store, mangas)

	// could be a markup change on the site, try again next time
	if len(mangas) == 0 {
		return mangas, nil
	}
	return mangas, store.SetMangas(first.key(), mangas)
}

// withCachedMetadata: The mangas whose metadata was read before (MangaByID, MangaVolumes) get it.
func (s *Scraper) withCachedMetadata(store mango.Store, mangas []mangadata.Manga) {
	for _, manga := range mangas {
		if m, ok := manga.(*mango.Manga); ok {
			s.cachedMetadata(store, m)
		}
	}
}

// searchRequests: The request of the search's first page, and of page n if the pages are
// built (nil if they're the next links).
func (s *Scraper) searchRequests(query string) (Request, func(n int) (Request, error), error) {
	if s.config.SearchRequest != nil {
		first, err := s.config.SearchRequest(s.config.BaseURL, query, 1)
		return first, func(n int) (Request, error) { return s.config.SearchRequest(s.config.BaseURL, query, n) }, err
	}
	searchURL, err := s.config.GenerateSearchURL(s.config.BaseURL, query)
	if err != nil {
		return Request{}, nil, err
	}
	if p := s.config.SearchPagination; p == nil || p.URL == nil {
		return Request{URL: searchURL}, nil, nil
	}
	return Request{URL: searchURL}, func(n int) (Request, error) {
		u, err := s.config.SearchPagination.URL(searchURL, n)
		return Request{URL: u}, err
	}, nil
}

// search: The mangas in the search's pages, see Pagination. pageN is page n's request, nil
// to follow the NextSelector's links.
func (s *Scraper) search(ctx context.Context, first Request, pageN func(n int) (Request, error)) ([]mangadata.Manga, error) {
	pagination := s.config.SearchPagination
	maxPages := 1
	if pagination != nil && (pagination.NextSelector != "" || pageN != nil) {
		maxPages = pagination.MaxPages
		if maxPages <= 0 {
			maxPages = defaultMaxPages
		}
	}

	var mangas []mangadata.Manga
	seen := map[string]bool{}
	request := first
	for n := 1; ; n++ {
		if n > 1 {
			s.log.Progress(ctx, n, maxPages, "%d results so far, search page", len(mangas))
		}
		var hasNext bool
		var nextURL string
		err := s.scrapeRequest(s.mangaAction(ctx), request, s.config.BaseURL, func(p *page) error {
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
			if n >= maxPages || !hasNext {
				return nil
			}
			if pagination.PageSize > 0 {
				// skipped ones (no id) count, the site sent them
				hasNext = p.doc.Find(s.config.MangaExtractor.Selector).Length() >= pagination.PageSize
				return nil
			}
			if pagination.NextSelector == "" {
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
		if pageN != nil {
			if request, err = pageN(n + 1); err != nil {
				return nil, err
			}
		} else if nextURL != "" {
			request = Request{URL: nextURL}
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
	if meta, ok := s.ownMetadata(manga); ok {
		if err := s.cacheMetadata(store, manga, meta); err != nil {
			return nil, false, err
		}
	}
	return manga, true, store.SetMangas(cacheID, []mangadata.Manga{manga})
}

var errorPagePath = regexp.MustCompile(`^/(400|404)/?$`)

// parseManga: The manga in its page with its metadata (see MetadataExtractor), nil if it isn't
// there (or has no title).
func (s *Scraper) parseManga(p *page, id string) *mango.Manga {
	selection := p.doc.Find(s.config.MangaByIDExtractor.Selector).First()
	if selection.Length() == 0 {
		return nil
	}
	title := mango.CleanString(s.config.MangaByIDExtractor.Title(selection))
	if title == "" {
		return nil
	}
	var meta metadata.Metadata = &mangadata.Metadata{}
	manga := &mango.Manga{
		Title:     title,
		URL:       p.url.String(),
		ID:        id,
		Cover:     s.config.MangaByIDExtractor.Cover(selection),
		Metadata_: &meta,
	}
	if s.config.MetadataExtractor != nil {
		setMetadata(manga, s.extractMetadata(p, manga))
	}
	return manga
}

// parseMangas: The mangas in a search page, without their metadata (see MetadataExtractor).
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
