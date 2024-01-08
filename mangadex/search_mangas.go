package mangadex

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangodex"
	"github.com/luevano/mangoprovider/mango"
	"github.com/philippgille/gokv"
)

func (d *Dex) SearchMangas(ctx context.Context, store gokv.Store, query string) ([]libmangal.Manga, error) {
	var mangas []libmangal.Manga

	params := url.Values{}
	params.Set("title", query)
	params.Set("limit", strconv.Itoa(100))
	params.Set("order[followedCount]", mangodex.OrderDescending)
	params.Add("includes[]", string(mangodex.RelationshipTypeCoverArt))

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
	// TODO: use incoming options instead of checking for empty
	if language == "" {
		language = "en"
	}

	for _, manga := range mangaList {
		mangaTitle := manga.GetTitle(language)
		mangaID := manga.ID

		var mangaCoverFileNames []string
		for _, relationship := range manga.Relationships {
			if relationship.Type == mangodex.RelationshipTypeCoverArt {
				coverRel, ok := relationship.Attributes.(*mangodex.CoverAttributes)
				if !ok {
					return nil, fmt.Errorf("unexpected error, failed to convert relationship attribute to cover type despite being of type %q", mangodex.RelationshipTypeCoverArt)
				}
				mangaCoverFileNames = append(mangaCoverFileNames, coverRel.FileName)
			}
		}

		var cover string
		if len(mangaCoverFileNames) != 0 {
			cover = fmt.Sprintf("https://mangadex.org/covers/%s/%s", mangaID, mangaCoverFileNames[0])
		}

		m := mango.MangoManga{
			Title:         mangaTitle,
			AnilistSearch: mangaTitle,
			URL:           fmt.Sprintf("https://mangadex.org/title/%s", mangaID),
			ID:            mangaID,
			Cover:         cover,
		}

		mangas = append(mangas, m)
	}

	err = store.Set(queryWithParams, mangas)
	if err != nil {
		return nil, err
	}

	return mangas, nil
}
