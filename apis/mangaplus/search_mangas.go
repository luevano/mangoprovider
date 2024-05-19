package mangaplus

import (
	"context"
	"fmt"
	"strconv"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/luevano/libmangal"
	"github.com/luevano/mangoplus"
	mango "github.com/luevano/mangoprovider"
	"github.com/philippgille/gokv"
)

func (p *plus) SearchMangas(ctx context.Context, store gokv.Store, query string) ([]libmangal.Manga, error) {
	var mangas []libmangal.Manga

	cacheID := fmt.Sprintf("%s-%s-%s", query, p.filter.Language, p.filter.MangaPlusQuality)

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

	// Will default to english
	prefLang := mangoplus.StringToLanguage(p.filter.Language)
	for _, manga := range mangaList {
		for _, title := range manga.Titles {
			// Sometimes when the language is not provided
			// it's because it's the english one
			lang := title.Language
			if lang == nil || lang == &prefLang {
				if fuzzy.MatchNormalizedFold(query, title.Name) {
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
	}

	err = store.Set(cacheID, mangas)
	if err != nil {
		return nil, err
	}

	return mangas, nil
}
