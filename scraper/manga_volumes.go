package scraper

import (
	"context"
	"fmt"

	"github.com/gocolly/colly/v2"
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/philippgille/gokv"
)

func (s *Scraper) MangaVolumes(_ctx context.Context, store gokv.Store, manga mango.Manga) ([]libmangal.Volume, error) {
	var volumes []libmangal.Volume

	// need an identifiable string for the cache
	cacheID := fmt.Sprintf("%s-volumes", manga.URL)

	found, err := store.Get(cacheID, &volumes)
	if err != nil {
		return nil, err
	}
	if found {
		mango.Log(fmt.Sprintf("found volumes in cache for manga %q with id %q", manga.Title, manga.ID))
		return volumes, nil
	}

	ctx := colly.NewContext()
	ctx.Put("manga", manga)
	ctx.Put("volumes", &volumes)

	err = s.volumesCollector.Request("GET", manga.URL, nil, ctx, nil)
	if err != nil {
		return nil, err
	}
	s.volumesCollector.Wait()

	err = store.Set(cacheID, volumes)
	if err != nil {
		return nil, err
	}

	return volumes, nil
}
