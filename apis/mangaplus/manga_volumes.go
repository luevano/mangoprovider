package mangaplus

import (
	"context"

	"github.com/luevano/libmangal/mangadata"
	mango "github.com/luevano/mangoprovider"
)

func (p *plus) MangaVolumes(ctx context.Context, store mango.Store, manga mango.Manga) ([]mangadata.Volume, error) {
	// MangoPlus doesn't provide volume information
	return []mangadata.Volume{
		&mango.Volume{Number: 1, None: true, Manga_: &manga},
	}, nil
}
