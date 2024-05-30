package mangadex

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"

	"github.com/luevano/libmangal/mangadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/philippgille/gokv"
)

func (d *dex) MangaVolumes(ctx context.Context, store gokv.Store, manga mango.Manga) ([]mangadata.Volume, error) {
	var volumes []mangadata.Volume

	params := url.Values{}
	params.Add("translatedLanguage[]", d.filter.Language)

	cacheID := fmt.Sprintf("%s?%s-%s", manga.ID, params.Encode(), d.filter.String())

	found, err := store.Get(cacheID, &volumes)
	if err != nil {
		return nil, err
	}
	if found {
		mango.Log("found volumes in cache for manga %q", manga.String())
		return volumes, nil
	}

	volumeList, err := d.client.Volume.List(manga.ID, params)
	if err != nil {
		return nil, err
	}

	none := false
	latestNumber := float32(0.0)
	// n is a string, could be "none", represents the volume number
	for n := range volumeList {
		if n == "none" {
			none = true
			continue
		}

		number64, err := strconv.ParseFloat(n, 32)
		if err != nil {
			return nil, err
		}
		number := float32(number64)
		if number > latestNumber {
			latestNumber = number
		}

		v := mango.Volume{
			Number: number,
			Manga_: &manga,
		}
		volumes = append(volumes, &v)
	}

	if none {
		volumes = append(volumes, &mango.Volume{
			Number: float32(int(latestNumber + float32(1.0))),
			None:   none,
			Manga_: &manga,
		})
	}

	sort.SliceStable(volumes, func(i, j int) bool {
		return volumes[i].Info().Number < volumes[j].Info().Number
	})

	err = store.Set(cacheID, volumes)
	if err != nil {
		return nil, err
	}

	return volumes, nil
}
