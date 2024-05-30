package mangapluscreators

import (
	"context"
	"fmt"

	"github.com/luevano/libmangal/mangadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/philippgille/gokv"
)

func (c *mpc) SearchMangas(ctx context.Context, store gokv.Store, query string) ([]mangadata.Manga, error) {
	var mangas []mangadata.Manga

	matchGroups := mango.ReNamedGroups(mango.MangaQueryIDRegex, query)
	_, byID := matchGroups[mango.MangaQueryIDName]
	if byID {
		return nil, fmt.Errorf("MangaPlusCreators doesn't support search manga by id")
	}

	cacheID := fmt.Sprintf("%s-%s-%s", query, c.filter.Language, c.filter.MangaPlusQuality)
	found, err := store.Get(cacheID, &mangas)
	if err != nil {
		return nil, err
	}
	if found {
		mango.Log("found mangas in cache for query %q", query)
		return mangas, nil
	}

	err = c.searchMangas(&mangas, query)
	if err != nil {
		return nil, err
	}

	err = store.Set(cacheID, mangas)
	if err != nil {
		return nil, err
	}

	return mangas, nil
}

func (c *mpc) searchMangas(mangas *[]mangadata.Manga, query string) error {
	page := 1
	for {
		// Will default to english or the only available language
		mangasDTO, err := c.client.Manga.List(query, c.filter.Language, page)
		if err != nil {
			return err
		}
		pagination := mangasDTO.Pagination
		if pagination == nil {
			return fmt.Errorf("unexpected error: titlesDto is nil for query %q, page %d", query, page)
		}

		for _, manga := range *mangasDTO.TitleList {
			m := &mango.Manga{
				Title:         manga.Title,
				AnilistSearch: manga.Title,
				URL:           fmt.Sprintf("%stitles/%s", website, manga.TitleID),
				ID:            manga.TitleID,
				Cover:         manga.ThumbnailURL,
			}
			*mangas = append(*mangas, m)
		}
		if !pagination.HasNextPage() {
			break
		}
		// TODO: need to double check this
		page += 1
	}
	return nil
}
