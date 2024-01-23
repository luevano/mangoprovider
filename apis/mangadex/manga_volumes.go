package mangadex

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"

	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/philippgille/gokv"
)

func (d *dex) MangaVolumes(ctx context.Context, store gokv.Store, manga mango.Manga) ([]libmangal.Volume, error) {
	var volumes []libmangal.Volume

	params := url.Values{}
	params.Add("translatedLanguage[]", d.filter.Language)

	cacheID := fmt.Sprintf("%s?%s-%s", manga.ID, params.Encode(), d.filter.String())

	found, err := store.Get(cacheID, &volumes)
	if err != nil {
		return nil, err
	}
	if found {
		mango.Log(fmt.Sprintf("[%s]found volumes in cache for manga %q with id %q", providerInfo.ID, manga.Title, manga.ID))
		return volumes, nil
	}

	volumeList, err := d.client.Volume.List(manga.ID, params)
	if err != nil {
		return nil, err
	}

	// represents the "none" volume
	var noneVolume mango.Volume

	// n is a string, could be "none", represents the volume number
	for n := range volumeList {
		// Using 0 for the "none"; shouldn't be used according to libmangal
		number := 0
		if n != "none" {
			numberI, err := strconv.Atoi(n)
			if err != nil {
				return nil, err
			}
			number = numberI
		}

		v := mango.Volume{
			Number: number,
			Manga_: &manga,
		}
	
		if number == 0 {
			noneVolume = v
		} else {
			volumes = append(volumes, v)
		}
	}

	sort.Slice(volumes, func(i, j int) bool {
		return volumes[i].Info().Number < volumes[j].Info().Number
	})

	if noneVolume != (mango.Volume{}) {
		volumes = append(volumes, noneVolume)
	}

	err = store.Set(cacheID, volumes)
	if err != nil {
		return nil, err
	}

	return volumes, nil
}
