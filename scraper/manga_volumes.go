package scraper

import (
	"context"
	"fmt"
	"sort"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/luevano/libmangal/mangadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/headless/rod"
)

func (s *Scraper) MangaVolumes(_ctx context.Context, store mango.Store, manga mango.Manga) ([]mangadata.Volume, error) {
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

	ctx := colly.NewContext()
	ctx.Put("manga", manga)
	ctx.Put("volumes", &volumes)

	collector := s.getVolumesCollector()
	err = collector.Request("GET", manga.URL, nil, ctx, nil)
	if err != nil {
		return nil, err
	}
	collector.Wait()

	sort.SliceStable(volumes, func(i, j int) bool {
		return volumes[i].Info().Number < volumes[j].Info().Number
	})

	err = store.SetVolumes(cacheID, volumes)
	if err != nil {
		return nil, err
	}

	return volumes, nil
}

// Get the volumes collector, the actual scraping logic is defined here.
func (s *Scraper) getVolumesCollector() *colly.Collector {
	collector := s.collector.Clone()
	s.setCollectorOnRequest(collector, s.config, rod.ActionVolume)
	collector.OnHTML("html", func(e *colly.HTMLElement) {
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
