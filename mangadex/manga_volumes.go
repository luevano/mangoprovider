package mangadex

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/mango"
	"github.com/philippgille/gokv"
)

func (d *dex) MangaVolumes(ctx context.Context, store gokv.Store, manga mango.Manga) ([]libmangal.Volume, error) {
	var volumes []libmangal.Volume

	language := d.options.Language
	// TODO: use incoming options instead of checking for empty
	if language == "" {
		language = "en"
	}
	params := url.Values{}
	params.Set("translatedLanguage[]", language)

	// need an identifiable string for the cache, this is not actually the query/url
	idWithParams := fmt.Sprintf("%s?%s", manga.ID, params.Encode())
	found, err := store.Get(idWithParams, &volumes)
	if err != nil {
		return nil, err
	}
	if found {
		// TODO: use logger
		// fmt.Printf("found volumes in cache for manga %q with id %q\n", manga.Title, manga.ID)
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

	err = store.Set(idWithParams, volumes)
	if err != nil {
		return nil, err
	}

	return volumes, nil
}
