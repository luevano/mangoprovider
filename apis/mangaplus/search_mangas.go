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

	matchGroups := mango.ReNamedGroups(mango.MangaQueryIDRegex, query)
	mangaID, byID := matchGroups[mango.MangaQueryIDName]

	var cacheID string
	if byID {
		cacheID = fmt.Sprintf("mid:%s", mangaID)
	} else {
		cacheID = fmt.Sprintf("%s-%s-%s", query, p.filter.Language, p.filter.MangaPlusQuality)
	}

	found, err := store.Get(cacheID, &mangas)
	if err != nil {
		return nil, err
	}
	if found {
		mango.Log(fmt.Sprintf("Found mangas in cache (%s)", query))
		return mangas, nil
	}

	if byID {
		err = p.searchManga(&mangas, mangaID)
	} else {
		err = p.searchMangas(&mangas, query)
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

func (p *plus) searchManga(mangas *[]libmangal.Manga, id string) error {
	titleDetail, err := p.client.Manga.Get(id)
	if err != nil {
		return err
	}
	*mangas = []libmangal.Manga{p.plusToMangoManga(titleDetail.Title)}
	return nil
}

func (p *plus) searchMangas(mangas *[]libmangal.Manga, query string) error {
	mangaList, err := p.client.Manga.All()
	if err != nil {
		return err
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
					*mangas = append(*mangas, p.plusToMangoManga(title))
				}
			}
		}
	}
	return nil
}

func (p *plus) plusToMangoManga(title mangoplus.Title) *mango.Manga {
	return &mango.Manga{
		Title:         title.Name,
		AnilistSearch: title.Name,
		URL:           fmt.Sprintf("%stitles/%d", website, title.TitleID),
		ID:            strconv.Itoa(title.TitleID),
		Cover:         title.PortraitImageURL,
	}
}
