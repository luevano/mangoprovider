package mangadex

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/luevano/mangodex"
	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/mango"
	"github.com/philippgille/gokv"
)

func (d *Dex) SearchMangas(ctx context.Context, store gokv.Store, query string) ([]libmangal.Manga, error) {
	var mangas []libmangal.Manga

	params := url.Values{}
	params.Set("limit", strconv.Itoa(100))
	params.Set("order[followedCount]", "desc")
	params.Set("title", query)

	ratings := []mangodex.ContentRating{mangodex.ContentRatingSafe, mangodex.ContentRatingSuggestive}
	if d.options.NSFW {
		ratings = append(ratings, mangodex.ContentRatingPorn)
		ratings = append(ratings, mangodex.ContentRatingErotica)
	}

	for _, rating := range ratings {
		params.Add("contentRating[]", string(rating))
	}

	// need a query with params included for the cache
	queryWithParams := params.Encode()

	found, err := store.Get(queryWithParams, &mangas)
	if err != nil {
		return nil, err
	}
	if found {
		// TODO: use logger
		// fmt.Printf("found mangas in cache with query %q\n", query)
		return mangas, nil
	}

	mangaList, err := d.client.Manga.List(params)
	if err != nil {
		// TODO: need to start using the logger, need to receive a logger, check options
		// log.Fatalln(err)
		return nil, err
	}

	language := d.options.Language
	if language == "" {
		language = "en"
	}

	for _, manga := range mangaList {
		mangaTitle := manga.GetTitle(language)
		mangaID := manga.ID
		m := mango.MangoManga{
			Title:         mangaTitle,
			AnilistSearch: mangaTitle,
			URL:           fmt.Sprintf("https://mangadex.org/title/%s", mangaID),
			ID:            mangaID,
			// TODO: need to implement the cover_art (needs to be added to mangodex)
		}

		mangas = append(mangas, m)
	}

	err = store.Set(queryWithParams, mangas)
	if err != nil {
		return nil, err
	}

	return mangas, nil
}
