package mangapluscreators

import (
	"context"
	"fmt"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	"github.com/luevano/mangoplus/creators"
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

	cacheID := fmt.Sprintf("%s-%s", query, c.filter.Language)
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
		titlesDTO, err := c.client.Manga.List(query, c.filter.Language, page)
		if err != nil {
			return err
		}
		pagination := titlesDTO.Pagination
		if pagination == nil {
			return fmt.Errorf("unexpected error: pagination is nil for query %q, page %d", query, page)
		}

		for _, title := range *titlesDTO.TitleList {
			*mangas = append(*mangas, c.mpcToMangoManga(title))
		}
		if !pagination.HasNextPage() {
			break
		}
		// TODO: need to double check this
		page += 1
	}
	return nil
}

func (c *mpc) mpcToMangoManga(title creators.Title) *mango.Manga {
	metadata := c.mpcToMetadata(title)
	return &mango.Manga{
		Title:         metadata.Title(),
		AnilistSearch: metadata.Title(),
		URL:           metadata.URL,
		ID:            title.TitleID,
		Cover:         metadata.CoverImage,
		Metadata_:     metadata,
	}
}

func (c *mpc) mpcToMetadata(title creators.Title) *metadata.Metadata {
	// TODO: decide if an id should be parsed from the provided string,
	// usually coming with "fm" in front
	//
	// Assume it is in english, there are no alternate titles
	// There is really not much info...
	return &metadata.Metadata{
		EnglishTitle:   title.Title,
		Description:    title.Description,
		CoverImage:     title.ThumbnailURL,
		Authors:        []string{title.HandleName},
		Artists:        []string{title.HandleName},
		StartDate:      parseTSMilli(title.FirstPublishDate),
		Status:         metadata.Status("UNKNOWN"), // mpc doesn't provide a status
		URL:            fmt.Sprintf("%stitles/%s", website, title.TitleID),
		IDProviderName: "mpc",
	}
}
