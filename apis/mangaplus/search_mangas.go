package mangaplus

import (
	"context"
	"fmt"
	"strconv"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	"github.com/luevano/mangoplus"
	mango "github.com/luevano/mangoprovider"
)

func (p *plus) SearchMangas(ctx context.Context, store mango.Store, query string) ([]mangadata.Manga, error) {
	var mangas []mangadata.Manga

	matchGroups := mango.ReNamedGroups(mango.MangaQueryIDRegex, query)
	mangaID, byID := matchGroups[mango.MangaQueryIDName]

	var cacheID string
	if byID {
		cacheID = fmt.Sprintf("mid:%s", mangaID)
	} else {
		cacheID = fmt.Sprintf("%s-%s-%s", query, p.filter.Language, p.options.Quality)
	}

	found, err := store.GetMangas(cacheID, query, &mangas)
	if err != nil {
		return nil, err
	}
	if found {
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

	err = store.SetMangas(cacheID, mangas)
	if err != nil {
		return nil, err
	}

	return mangas, nil
}

func (p *plus) searchManga(mangas *[]mangadata.Manga, id string) error {
	titleDetail, err := p.client.Manga.Get(id)
	if err != nil {
		return err
	}
	*mangas = []mangadata.Manga{p.plusToMangoManga(titleDetail)}
	return nil
}

func (p *plus) searchMangas(mangas *[]mangadata.Manga, query string) error {
	titlesGroupList, err := p.client.Manga.All()
	if err != nil {
		return err
	}

	// Will default to english
	prefLang := mangoplus.StringToLanguage(p.filter.Language)
	for _, titleGroup := range titlesGroupList {
		for _, title := range titleGroup.Titles {
			// Sometimes when the language is not provided
			// it's because it's the english one
			lang := title.Language
			if lang == nil || lang == &prefLang {
				if fuzzy.MatchNormalizedFold(query, title.Name) {
					// Search for the titleDetail details and use that instead (contains more data)
					titleDetail, err := p.client.Manga.Get(strconv.Itoa(title.TitleID))
					if err != nil {
						return err
					}
					*mangas = append(*mangas, p.plusToMangoManga(titleDetail))
				}
			}
		}
	}
	return nil
}

func (p *plus) plusToMangoManga(titleDetail mangoplus.TitleDetailView) *mango.Manga {
	metadata := p.plusToMetadata(titleDetail)
	return &mango.Manga{
		Title:         metadata.Title(),
		AnilistSearch: metadata.Title(),
		URL:           metadata.URL,
		ID:            strconv.Itoa(titleDetail.Title.TitleID),
		Cover:         metadata.CoverImage,
		Banner:        metadata.BannerImage,
		Metadata_:     metadata,
	}
}

func (p *plus) plusToMetadata(titleDetail mangoplus.TitleDetailView) *metadata.Metadata {
	title := titleDetail.Title

	var status metadata.Status
	switch titleDetail.TitleLabels.ReleaseSchedule {
	case mangoplus.ReleaseScheduleCompleted:
		status = metadata.StatusFinished
	// need to check if this is what disabled is used for,
	// maybe it's for cancelled or not yet released?
	case mangoplus.ReleaseScheduleDisabled:
		status = metadata.StatusHiatus
	default:
		status = metadata.StatusReleasing

	}
	// Only way to look at the start date is by looking at the very first chapter,
	// but there is no guarantee that the date of the first chapter is that of the
	// manga start date, as the chapter could've been released much later in time
	var date metadata.Date
	for _, chapterListGroup := range titleDetail.ChapterListGroup {
		for _, chapter := range chapterListGroup.FirstChapterList {
			date = parseTSSecs(chapter.StartTimeStamp)
			break
		}
		break
	}

	// Assume it is in english, there are no alternate titles (unless many other requests where to be performed).
	// Also using author as aritsts too, not much data provided, and there is no tag/genre data
	return &metadata.Metadata{
		EnglishTitle:   title.Name,
		Description:    titleDetail.Overview,
		CoverImage:     title.PortraitImageURL,
		BannerImage:    titleDetail.TitleImageUrl,
		Authors:        []string{title.Author},
		Artists:        []string{title.Author},
		StartDate:      date,
		Status:         status,
		URL:            fmt.Sprintf("%stitles/%d", website, title.TitleID),
		IDProvider:     title.TitleID,
		IDProviderName: "mp",
	}
}
