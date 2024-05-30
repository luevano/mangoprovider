package mangadex

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/mangodex"
	mango "github.com/luevano/mangoprovider"
	"github.com/philippgille/gokv"
)

func (d *dex) SearchMangas(ctx context.Context, store gokv.Store, query string) ([]mangadata.Manga, error) {
	var mangas []mangadata.Manga

	matchGroups := mango.ReNamedGroups(mango.MangaQueryIDRegex, query)
	mangaID, byID := matchGroups[mango.MangaQueryIDName]

	limit := 100
	var params url.Values
	var cacheID string
	if byID {
		cacheID = fmt.Sprintf("mid:%s", mangaID)
	} else {
		params = d.getSearchMangasParams(query, limit)
		cacheID = fmt.Sprintf("%s-%s", params.Encode(), d.filter.String())
	}

	found, err := store.Get(cacheID, &mangas)
	if err != nil {
		return nil, err
	}
	if found {
		mango.Log("found mangas in cache for query %q", query)
		return mangas, nil
	}

	if byID {
		err = d.searchManga(&mangas, mangaID)
	} else {
		err = d.searchMangas(&mangas, params, limit)
	}
	if err != nil {
		return nil, err
	}

	err = store.Set(cacheID, mangas)
	if err != nil {
		return nil, err
	}

	return mangas, nil
}

// find manga by id and populate the list
func (d *dex) searchManga(mangas *[]mangadata.Manga, id string) error {
	params := url.Values{}
	params.Add("includes[]", string(mangodex.RelationshipTypeCoverArt))
	manga, err := d.client.Manga.Get(id, params)
	if err != nil {
		return err
	}
	*mangas = []mangadata.Manga{d.dexToMangoManga(manga)}
	return nil
}

// search of mangas by query and populate the list
func (d *dex) searchMangas(mangas *[]mangadata.Manga, params url.Values, limit int) error {
	offset := 0
	for {
		params.Set("offset", strconv.Itoa(offset))
		mangaList, err := d.client.Manga.List(params)
		if err != nil {
			return err
		}
		for _, manga := range mangaList {
			*mangas = append(*mangas, d.dexToMangoManga(manga))
		}
		// If received less than the limit, we got them all
		if len(mangaList) < limit {
			break
		}
		offset += limit
	}
	return nil
}

func (d *dex) getSearchMangasParams(query string, limit int) url.Values {
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

	return params
}

func (d *dex) dexToMangoManga(manga *mangodex.Manga) *mango.Manga {
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
	return &mango.Manga{
		Title:         mangaTitle,
		AnilistSearch: mangaTitle,
		URL:           fmt.Sprintf("%stitle/%s", website, manga.ID),
		ID:            manga.ID,
		Cover:         cover,
	}
}
