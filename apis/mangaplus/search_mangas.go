package mangaplus

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/philippgille/gokv"
)

func (p *plus) SearchMangas(ctx context.Context, store gokv.Store, query string) ([]libmangal.Manga, error) {
	var mangas []libmangal.Manga

	// TODO: include other options such as image quality
	cacheID := query

	found, err := store.Get(cacheID, &mangas)
	if err != nil {
		return nil, err
	}
	if found {
		mango.Log(fmt.Sprintf("Found mangas in cache with query %q", query))
		return mangas, nil
	}

	mangaList, err := p.client.Manga.All()
	if err != nil {
		return nil, err
	}

	sanQuery := strings.TrimSpace(strings.ToLower(query))
	for _, manga := range mangaList {
		// TODO: use helper method once implemented
		//
		// The main title is probably the english one
		// mangaTitle := manga.TheTitle

		// TODO: actually select the manga that best matches
		for _, title := range manga.Titles {
			currTitle := strings.TrimSpace(strings.ToLower(title.Name))
			if strings.Contains(currTitle, sanQuery) {
				m := mango.Manga{
					Title:         title.Name,
					AnilistSearch: title.Name,
					URL:           fmt.Sprintf("%stitles/%d", website, title.TitleID),
					ID:            strconv.Itoa(title.TitleID),
					Cover:         title.PortraitImageURL,
				}

				mangas = append(mangas, &m)
			}
		}
	}

	err = store.Set(cacheID, mangas)
	if err != nil {
		return nil, err
	}

	return mangas, nil
}
