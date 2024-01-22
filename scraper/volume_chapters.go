package scraper

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/gocolly/colly/v2"
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/philippgille/gokv"
)

func (s *Scraper) VolumeChapters(_ctx context.Context, logger *libmangal.Logger, store gokv.Store, volume mango.Volume) ([]libmangal.Chapter, error) {
	var chapters []libmangal.Chapter

	// need an identifiable string for the cache
	cacheID := fmt.Sprintf("%s-chapters", volume.Manga_.URL)

	found, err := store.Get(cacheID, &chapters)
	if err != nil {
		return nil, err
	}
	if found {
		logger.Log(fmt.Sprintf("[%s]found chapters in cache for manga %q with id %q", s.options.Name, volume.Manga_.Title, volume.Manga_.ID))
		return chapters, nil
	}

	ctx := colly.NewContext()
	ctx.Put("volume", volume)
	ctx.Put("chapters", &chapters)

	// TODO: check if using this URL is good enough, only works for sources that
	// don't provide volumes and thus everything is in the manga url
	err = s.chaptersCollector.Request(http.MethodGet, volume.Manga_.URL, nil, ctx, nil)
	if err != nil {
		return nil, err
	}
	s.chaptersCollector.Wait()

	if s.options.ReverseChapters {
		slices.Reverse(chapters)
	}

	// TODO: only cache if there are chapters (len > 0)?
	err = store.Set(cacheID, chapters)
	if err != nil {
		return nil, err
	}

	return chapters, nil
}