package scraper

import (
	"context"
	"fmt"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/mango"
	"github.com/philippgille/gokv"
)

func (s *Scraper) VolumeChapters(_ctx context.Context, store gokv.Store, volume mango.Volume) ([]libmangal.Chapter, error) {
	// TODO: use gokv.Store
	// if chapters := s.cache.chapters.Get(manga.URL); chapters.IsPresent() {
	// 	c := chapters.MustGet()
	// 	for _, chapter := range c {
	// 		chapter.Manga = manga
	// 	}
	// 	manga.Chapters = c

	// 	return nil
	// }

	// ctx := colly.NewContext()
	// ctx.Put("volume", volume)
	// // TODO: check if volume.Manga_.URL is the one required
	// err := s.chaptersCollector.Request(http.MethodGet, volume.Manga_.URL, nil, ctx, nil)

	// if err != nil {
	// 	return nil, err
	// }

	// s.chaptersCollector.Wait()

	// if s.config.ReverseChapters {
	// 	// reverse chapters
	// 	chapters := manga.Chapters
	// 	reversed := make([]*source.Chapter, len(chapters))
	// 	for i, chapter := range chapters {
	// 		reversed[len(chapters)-i-1] = chapter
	// 		chapter.Index = uint16(len(chapters) - i - 1)
	// 		chapter.Index++
	// 	}

	// 	manga.Chapters = reversed
	// }

	// TODO: use gokv.Store
	// // Only cache if we have chapters
	// if len(manga.Chapters) > 0 {
	// 	_ = s.cache.chapters.Set(manga.URL, manga.Chapters)
	// }

	return nil, fmt.Errorf("unimplemented")
}
