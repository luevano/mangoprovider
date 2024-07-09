package mangapluscreators

import (
	"context"

	"github.com/luevano/libmangal/mangadata"
	mango "github.com/luevano/mangoprovider"
)

func (c *mpc) MangaVolumes(ctx context.Context, store mango.Store, manga mango.Manga) ([]mangadata.Volume, error) {
	// MangoPlusCreators doesn't provide volume information
	return []mangadata.Volume{
		&mango.Volume{Number: float32(1.0), Manga_: &manga},
	}, nil
}
