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
		mango.Log(fmt.Sprintf("found mangas in cache with query %q", query))
		return mangas, nil
	}

	mangaList, err := d.client.Manga.List(params)
	if err != nil {
		return nil, err
	}

	for _, manga := range mangaList {
		var mangaCoverFileNames []string
		for _, relationship := range manga.Relationships {
			if relationship.Type == mangodex.RelationshipTypeCoverArt {
				coverRel, _ := relationship.Attributes.(*mangodex.CoverAttributes)
				mangaCoverFileNames = append(mangaCoverFileNames, coverRel.FileName)
			}
		}

		var cover string
		if len(mangaCoverFileNames) != 0 {
			cover = fmt.Sprintf("https://mangadex.org/covers/%s/%s", manga.ID, mangaCoverFileNames[0])
		}

		mangaTitle := manga.GetTitle(d.filter.Language)
		m := mango.Manga{
			Title:         mangaTitle,
			AnilistSearch: mangaTitle,
			URL:           fmt.Sprintf("https://mangadex.org/title/%s", manga.ID),
			ID:            manga.ID,
			Cover:         cover,
		}

		mangas = append(mangas, &m)
	}

	err = store.Set(cacheID, mangas)
	if err != nil {
		return nil, err
	}

	return mangas, nil
}
