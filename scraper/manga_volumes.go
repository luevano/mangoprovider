package scraper

import (
	"context"
	"fmt"
	"sort"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/luevano/libmangal/mangadata"
	mango "github.com/luevano/mangoprovider"
)

func (s *Scraper) MangaVolumes(ctx context.Context, store mango.Store, manga mango.Manga) ([]mangadata.Volume, error) {
	var volumes []mangadata.Volume

	// need an identifiable string for the cache
	cacheID := fmt.Sprintf("%s-volumes", manga.URL)

	found, err := store.GetVolumes(cacheID, manga, &volumes)
	if err != nil {
		return nil, err
	}
	if found {
		return volumes, nil
	}

	// data shared with the collector callbacks
	data := colly.NewContext()
	data.Put("manga", manga)
	data.Put("volumes", &volumes)

	collector := s.getVolumesCollector()
	if err := scrape(ctx, collector, manga.URL, data); err != nil {
		return nil, err
	}

	sort.SliceStable(volumes, func(i, j int) bool {
		return volumes[i].Info().Number < volumes[j].Info().Number
	})

	// could be a markup change on the site, try again next time
	if len(volumes) == 0 {
		return volumes, nil
	}
	err = store.SetVolumes(cacheID, volumes)
	if err != nil {
		return nil, err
	}

	return volumes, nil
}

// Get the volumes collector, the actual scraping logic is defined here.
func (s *Scraper) getVolumesCollector() *colly.Collector {
	collector := s.collector.Clone()
	s.setCollectorOnRequest(collector, requestVolume)
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		defer recoverScrape(e)
		elements := e.DOM.Find(s.config.VolumeExtractor.Selector)
		manga := e.Request.Ctx.GetAny("manga").(mango.Manga)
		volumes := e.Request.Ctx.GetAny("volumes").(*[]mangadata.Volume)

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
