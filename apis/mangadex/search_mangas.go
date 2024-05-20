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
	limit := 100
	var mangas []libmangal.Manga

	params := url.Values{}
	params.Set("title", query)
	params.Set("limit", strconv.Itoa(limit))
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
		mango.Log(fmt.Sprintf("Found mangas in cache with query %q", query))
		return mangas, nil
	}

	offset := 0
	for {
		// The offset is set on each iteration, shouldn't be included in the cacheID.
		params.Set("offset", strconv.Itoa(offset))
		ended, err := d.populateMangas(&mangas, params)
		if err != nil {
			return nil, err
		}
		if ended {
			break
		}
		offset += limit
	}

	err = store.Set(cacheID, mangas)
	if err != nil {
		return nil, err
	}

	return mangas, nil
}

// Make the request and parse the responses, populating the manga list and extra info useful for filtering.
func (d *dex) populateMangas(mangas *[]libmangal.Manga, params url.Values) (bool, error) {
	mangaList, err := d.client.Manga.List(params)
	if err != nil {
		return false, err
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
			cover = fmt.Sprintf("%scovers/%s/%s", website, manga.ID, mangaCoverFileNames[0])
		}

		mangaTitle := manga.GetTitle(d.filter.Language)
		m := mango.Manga{
			Title:         mangaTitle,
			AnilistSearch: mangaTitle,
			URL:           fmt.Sprintf("%stitle/%s", website, manga.ID),
			ID:            manga.ID,
			Cover:         cover,
		}

		*mangas = append(*mangas, &m)
	}
	// If received 100 entries means it probably has more.
	if len(mangaList) == 100 {
		return false, nil
	}

	return true, nil
}
