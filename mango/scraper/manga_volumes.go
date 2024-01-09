package scraper

import (
	"context"
	"fmt"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/mango"
	"github.com/philippgille/gokv"
)

func (s *Scraper) MangaVolumes(_ctx context.Context, store gokv.Store, manga mango.Manga) ([]libmangal.Volume, error) {
	return nil, fmt.Errorf("unimplemented")
}
