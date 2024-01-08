package mangadex

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/mango"
	"github.com/philippgille/gokv"
)

func (d *Dex) MangaVolumes(ctx context.Context, store gokv.Store, manga mango.MangoManga) ([]libmangal.Volume, error) {
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

		v := mango.MangoVolume{
			Number: number,
			Manga_: &manga,
		}
		volumes = append(volumes, v)
	}

	err = store.Set(idWithParams, volumes)
	if err != nil {
		return nil, err
	}

	return volumes, nil
}
