package mangapluscreators

import (
	"context"
	"fmt"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	"github.com/luevano/mangoplus/creators"
	mango "github.com/luevano/mangoprovider"
)

func (c *mpc) SearchMangas(ctx context.Context, store mango.Store, query string) ([]mangadata.Manga, error) {
	var mangas []mangadata.Manga

	matchGroups := mango.ReNamedGroups(mango.MangaQueryIDRegex, query)
	_, byID := matchGroups[mango.MangaQueryIDName]
	if byID {
		return nil, fmt.Errorf("MangaPlusCreators doesn't support search manga by id")
	}

	cacheID := fmt.Sprintf("%s-%s", query, c.filter.Language)
	found, err := store.GetMangas(cacheID, query, &mangas)
	if err != nil {
		return nil, err
	}
	if found {
		return mangas, nil
	}

	err = c.searchMangas(&mangas, query)
	if err != nil {
		return nil, err
	}

	err = store.SetMangas(cacheID, mangas)
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
	meta := c.mpcToMetadata(title)
	var m metadata.Metadata = meta
	return &mango.Manga{
		Title:         meta.Title(),
		AnilistSearch: meta.Title(),
		URL:           meta.URL(),
		ID:            title.TitleID,
		Cover:         meta.Cover(),
		Metadata_:     &m,
	}
}

func (c *mpc) mpcToMetadata(title creators.Title) metadata.Metadata {
	// TODO: decide if an id should be parsed from the provided string,
	// usually coming with "fm" in front
	//
	// Assume it is in english, there are no alternate titles
	// There is really not much info...
	return &mangadata.Metadata{
		EnglishTitle:      title.Title,
		Summary:           title.Description,
		CoverImage:        title.ThumbnailURL,
		AuthorList:        []string{title.HandleName},
		ArtistList:        []string{title.HandleName},
		DateStart:         parseTSMilli(title.FirstPublishDate),
		PublicationStatus: metadata.Status("UNKNOWN"), // mpc doesn't provide a status
		SourceURL:         fmt.Sprintf("%stitles/%s", website, title.TitleID),
		ProviderIDCode:    "mpc",
	}
}
