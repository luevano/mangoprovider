package mangadex

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangodex"
	mango "github.com/luevano/mangoprovider"
	"github.com/philippgille/gokv"
)

func (d *dex) SearchMangas(ctx context.Context, store gokv.Store, query string) ([]libmangal.Manga, error) {
	var mangas []libmangal.Manga

	params := url.Values{}
	params.Set("title", query)
	params.Set("limit", strconv.Itoa(100))
	params.Set("order[followedCount]", mangodex.OrderDescending)
	params.Add("includes[]", string(mangodex.RelationshipTypeCoverArt))

	ratings := []mangodex.ContentRating{mangodex.ContentRatingSafe, mangodex.ContentRatingSuggestive}
	if d.filter.NSFW {
		ratings = append(ratings, mangodex.ContentRatingPorn)
		ratings = append(ratings, mangodex.ContentRatingErotica)
	}
	for _, rating := range ratings {
		params.Add("contentRating[]", string(rating))
	}

	cacheID := fmt.Sprintf("%s-%s", params.Encode(), d.filter.String())

	found, err := store.Get(cacheID, &mangas)
	if err != nil {
		return nil, err
	}
	if found {
		mango.Log(fmt.Sprintf("[%s]found mangas in cache with query %q", providerInfo.ID, query))
		return mangas, nil
	}

	mangaList, err := d.client.Manga.List(params)
	if err != nil {
		return nil, err
	}

	for _, manga := range mangaList {
		mangaTitle := manga.GetTitle(d.filter.Language)
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

		m := mango.Manga{
			Title:         mangaTitle,
			AnilistSearch: mangaTitle,
			URL:           fmt.Sprintf("https://mangadex.org/title/%s", mangaID),
			ID:            mangaID,
			Cover:         cover,
		}

		mangas = append(mangas, m)
	}

	err = store.Set(cacheID, mangas)
	if err != nil {
		return nil, err
	}

	return mangas, nil
}
