package scraper

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gocolly/colly/v2"
	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/mango"
	"github.com/philippgille/gokv"
)

func (s *Scraper) ChapterPages(_ctx context.Context, store gokv.Store, chapter mango.Chapter) ([]libmangal.Page, error) {
	var pages []libmangal.Page

	// need an identifiable string for the cache
	cacheID := fmt.Sprintf("%s-pages", chapter.URL)

	found, err := store.Get(cacheID, &pages)
	if err != nil {
		return nil, err
	}
	if found {
		// TODO: use logger
		// fmt.Printf("found volumes in cache for manga %q with id %q\n", manga.Title, manga.ID)
		return pages, nil
	}

	ctx := colly.NewContext()
	ctx.Put("chapter", chapter)
	ctx.Put("pages", &pages)

	err = s.pagesCollector.Request(http.MethodGet, chapter.URL, nil, ctx, nil)
	if err != nil {
		return nil, err
	}
	s.pagesCollector.Wait()

	err = store.Set(cacheID, pages)
	if err != nil {
		return nil, err
	}

	return pages, nil
}
