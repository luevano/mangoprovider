package scraper

import (
	"context"
	"fmt"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/mango"
	"github.com/philippgille/gokv"
)

func (s *Scraper) ChapterPages(_ctx context.Context, store gokv.Store, chapter mango.Chapter) ([]libmangal.Page, error) {

	// ctx := colly.NewContext()
	// ctx.Put("chapter", chapter)
	// err := s.pagesCollector.Request(http.MethodGet, chapter.URL, nil, ctx, nil)

	// if err != nil {
	// 	return err
	// }

	// s.pagesCollector.Wait()

	return nil, fmt.Errorf("unimplemented")
}
