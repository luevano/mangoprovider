package mangaplus

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	"github.com/luevano/mangoplus"
	mango "github.com/luevano/mangoprovider"
)

func (p *plus) SearchMangas(ctx context.Context, store mango.Store, query string) ([]mangadata.Manga, error) {
	if id, byID := mango.ReNamedGroups(mango.MangaQueryIDRegex, query)[mango.MangaQueryID]; byID {
		return mango.SearchByID(p.MangaByID(ctx, store, id))
	}

	var mangas []mangadata.Manga
	cacheID := fmt.Sprintf("%s-%s-%s", query, p.filter.Language, p.options.Quality)

	found, err := store.GetMangas(cacheID, query, &mangas)
	if err != nil {
		return nil, err
	}
	if found {
		return mangas, nil
	}

	if err := p.searchMangas(ctx, &mangas, query); err != nil {
		return nil, err
	}

	err = store.SetMangas(cacheID, mangas)
	if err != nil {
		return nil, err
	}

	return mangas, nil
}

// MangaByID gets the title by its number.
func (p *plus) MangaByID(ctx context.Context, store mango.Store, id string) (mangadata.Manga, bool, error) {
	cacheID := "mid:" + id
	var mangas []mangadata.Manga
	if found, err := store.GetMangas(cacheID, cacheID, &mangas); err != nil {
		return nil, false, err
	} else if found && len(mangas) == 1 {
		return mangas[0], true, nil
	}

	titleDetail, err := p.client.Manga.GetContext(ctx, id)
	if errors.Is(err, mangoplus.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	m := p.plusToMangoManga(titleDetail)
	return m, true, store.SetMangas(cacheID, []mangadata.Manga{m})
}

const maxSearchResults = 20

func (p *plus) searchMangas(ctx context.Context, mangas *[]mangadata.Manga, query string) error {
	// Will default to english
	prefLang := mangoplus.StringToLanguage(p.filter.Language)
	titles, err := p.client.Manga.AllContext(ctx, prefLang)
	if err != nil {
		return err
	}

	type match struct {
		title *mangoplus.Title
		rank  int // lower is a closer match
	}
	var matches []match
	for _, title := range titles {
		rank := fuzzy.RankMatchNormalizedFold(query, title.Name)
		if rank != -1 {
			matches = append(matches, match{title, rank})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].rank < matches[j].rank
	})

	// Each match needs its own request for the details, too slow for broad queries
	if len(matches) > maxSearchResults {
		p.log.Log("%d mangas found for %q, only using the best %d", len(matches), query, maxSearchResults)
		matches = matches[:maxSearchResults]
	}

	for _, m := range matches {
		// Search for the titleDetail details and use that instead (contains more data)
		titleDetail, err := p.client.Manga.GetContext(ctx, strconv.Itoa(int(m.title.TitleId)))
		if err != nil {
			return err
		}
		*mangas = append(*mangas, p.plusToMangoManga(titleDetail))
	}
	return nil
}

func (p *plus) plusToMangoManga(titleDetail *mangoplus.TitleDetailView) *mango.Manga {
	meta := p.plusToMetadata(titleDetail)
	var m metadata.Metadata = meta
	return &mango.Manga{
		Title:     meta.Title(),
		URL:       meta.URL(),
		ID:        strconv.Itoa(int(titleDetail.Title.TitleId)),
		Cover:     meta.Cover(),
		Banner:    meta.Banner(),
		Metadata_: &m,
	}
}

func (p *plus) plusToMetadata(titleDetail *mangoplus.TitleDetailView) *mangadata.Metadata {
	title := titleDetail.Title

	var status metadata.Status
	switch titleDetail.GetTitleLabels().GetReleaseSchedule() {
	case mangoplus.ReleaseSchedule_COMPLETED, mangoplus.ReleaseSchedule_ONE_SHOT:
		status = metadata.StatusFinished
	// need to check if this is what disabled is used for,
	// maybe it's for cancelled or not yet released?
	case mangoplus.ReleaseSchedule_DISABLED, mangoplus.ReleaseSchedule_HIATUS:
		status = metadata.StatusHiatus
	default:
		status = metadata.StatusReleasing
	}
	// Only way to look at the start date is by looking at the very first chapter,
	// but there is no guarantee that the date of the first chapter is that of the
	// manga start date, as the chapter could've been released much later in time
	var date metadata.Date
	if len(titleDetail.ChapterListV2) != 0 {
		date = parseTSSecs(titleDetail.ChapterListV2[0].StartTimeStamp)
	}

	// Assume it is in english, there are no alternate titles (unless many other requests where to be performed).
	// Also using author as aritsts too, not much data provided, and there is no tag/genre data
	return &mangadata.Metadata{
		EnglishTitle:      title.Name,
		Summary:           titleDetail.Overview,
		CoverImage:        title.PortraitImageUrl,
		BannerImage:       titleDetail.TitleImageUrl,
		AuthorList:        []string{title.Author},
		ArtistList:        []string{title.Author},
		DateStart:         date,
		PublicationStatus: status,
		SourceURL:         fmt.Sprintf("%stitles/%d", website, title.TitleId),
		ProviderID:        strconv.Itoa(int(title.TitleId)),
		ProviderIDCode:    "mp",
	}
}
