package scraper

import (
	"context"
	"fmt"

	"github.com/PuerkitoBio/goquery"
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

	collector := s.getVolumesCollector()
	err = collector.Request("GET", manga.URL, nil, ctx, nil)
	if err != nil {
		return nil, err
	}
	collector.Wait()

	err = store.Set(cacheID, volumes)
	if err != nil {
		return nil, err
	}

	return volumes, nil
}

// Get the volumes collector, the actual scraping logic is defined here.
func (s *Scraper) getVolumesCollector() *colly.Collector {
	collector := s.collector.Clone()
	setCollectorOnRequest(collector, s.config, "volume")
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		elements := e.DOM.Find(s.config.VolumeExtractor.Selector)
		manga := e.Request.Ctx.GetAny("manga").(mango.Manga)
		volumes := e.Request.Ctx.GetAny("volumes").(*[]libmangal.Volume)

		elements.Each(func(_ int, selection *goquery.Selection) {
			v := mango.Volume{
				Number: s.config.VolumeExtractor.Number(selection),
				Manga_: &manga,
			}
			*volumes = append(*volumes, &v)
		})
	})
	return collector
}
