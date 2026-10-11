package flamecomics

import (
	"context"

	"github.com/luevano/libmangal/mangadata"
	mango "github.com/luevano/mangoprovider"
)

func (f *flame) MangaVolumes(_ context.Context, _ mango.Store, manga mango.Manga) ([]mangadata.Volume, error) {
	// no volumes
	return []mangadata.Volume{&mango.Volume{Number: 1, None: true, Manga_: &manga}}, nil
}
