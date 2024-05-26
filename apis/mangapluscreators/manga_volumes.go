package mangapluscreators

import (
	"context"

	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/philippgille/gokv"
)

func (c *mpc) MangaVolumes(ctx context.Context, store gokv.Store, manga mango.Manga) ([]libmangal.Volume, error) {
	// MangoPlusCreators doesn't provide volume information
	return []libmangal.Volume{
		&mango.Volume{Number: float32(1.0), Manga_: &manga},
	}, nil
}
