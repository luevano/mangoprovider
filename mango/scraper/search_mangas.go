package scraper

import (
	"context"

	"github.com/gocolly/colly/v2"
	"github.com/luevano/libmangal"
	"github.com/philippgille/gokv"
)

func (s *Scraper) SearchMangas(_ctx context.Context, store gokv.Store, query string) ([]libmangal.Manga, error) {
	var mangas []libmangal.Manga

	searchURL, err := s.options.GenerateSearchURL(s.options.BaseURL, query)
	if err != nil {
		return nil, err
	}

	found, err := store.Get(searchURL, &mangas)
	if err != nil {
		return nil, err
	}
	if found {
		// TODO: use logger
		// fmt.Printf("found mangas in cache with query %q\n", query)
		return mangas, nil
	}

	ctx := colly.NewContext()
	ctx.Put("mangas", &mangas)

	err = s.mangasCollector.Request("GET", searchURL, nil, ctx, nil)
	if err != nil {
		return nil, err
	}
	s.mangasCollector.Wait()

	err = store.Set(searchURL, mangas)
	if err != nil {
		return nil, err
	}

	return mangas, nil
}
