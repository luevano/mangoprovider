package scraper

import (
	"context"
	"fmt"
	"sort"

	"github.com/PuerkitoBio/goquery"
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

	err = s.scrape(ctx, manga.URL, manga.URL, func(p *page) error {
		p.doc.Find(s.config.VolumeExtractor.Selector).Each(func(_ int, selection *goquery.Selection) {
			volumes = append(volumes, &mango.Volume{
				Number: s.config.VolumeExtractor.Number(selection),
				Manga_: &manga,
			})
		})
		return nil
	})
	if err != nil {
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
