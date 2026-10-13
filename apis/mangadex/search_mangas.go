package mangadex

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	"github.com/luevano/mangodex"
	mango "github.com/luevano/mangoprovider"
)

func (d *dex) SearchMangas(ctx context.Context, store mango.Store, query string) ([]mangadata.Manga, error) {
	if id, byID := mango.ReNamedGroups(mango.MangaQueryIDRegex, query)[mango.MangaQueryID]; byID {
		return mango.SearchByID(d.MangaByID(ctx, store, id))
	}

	var mangas []mangadata.Manga
	limit := 100
	params := d.getSearchMangasParams(query, limit)
	cacheID := fmt.Sprintf("%s-%s", params.Encode(), d.filter.String())

	found, err := store.GetMangas(cacheID, query, &mangas)
	if err != nil {
		return nil, err
	}
	if found {
		return mangas, nil
	}

	if err := d.searchMangas(ctx, &mangas, params, limit); err != nil {
		return nil, err
	}

	err = store.SetMangas(cacheID, mangas)
	if err != nil {
		return nil, err
	}

	return mangas, nil
}

// MangaByID gets the manga by its uuid.
func (d *dex) MangaByID(ctx context.Context, store mango.Store, id string) (mangadata.Manga, bool, error) {
	cacheID := "mid:" + id
	var mangas []mangadata.Manga
	if found, err := store.GetMangas(cacheID, cacheID, &mangas); err != nil {
		return nil, false, err
	} else if found && len(mangas) == 1 {
		return mangas[0], true, nil
	}

	params := url.Values{}
	params.Add("includes[]", string(mangodex.RelationshipTypeCoverArt))
	params.Add("includes[]", string(mangodex.RelationshipTypeAuthor))
	params.Add("includes[]", string(mangodex.RelationshipTypeArtist))
	manga, err := d.client.Manga.GetContext(ctx, id, params)
	if errors.Is(err, mangodex.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	m := d.dexToMangoManga(manga)
	return m, true, store.SetMangas(cacheID, []mangadata.Manga{m})
}

// searchPages caps the search (100 results each, the most followed first), a short query
// matches thousands of mangas (and MangaDex refuses an offset+limit over 10000)
const searchPages = 5

// search of mangas by query and populate the list
func (d *dex) searchMangas(ctx context.Context, mangas *[]mangadata.Manga, params url.Values, limit int) error {
	offset := 0
	for page := 1; page <= searchPages; page++ {
		if page > 1 {
			d.log.Progress(ctx, 0, 0, "%d results so far, getting page %d", len(*mangas), page)
		}
		params.Set("offset", strconv.Itoa(offset))
		mangaList, total, err := d.client.Manga.ListWithTotalContext(ctx, params)
		if err != nil {
			return err
		}
		for _, manga := range mangaList {
			*mangas = append(*mangas, d.dexToMangoManga(manga))
		}
		offset += limit
		// got them all
		if len(mangaList) < limit || offset >= total {
			break
		}
	}
	return nil
}

func (d *dex) getSearchMangasParams(query string, limit int) url.Values {
	params := url.Values{}
	params.Set("title", query)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("order[followedCount]", mangodex.OrderDescending)
	params.Add("includes[]", string(mangodex.RelationshipTypeCoverArt))
	params.Add("includes[]", string(mangodex.RelationshipTypeAuthor))
	params.Add("includes[]", string(mangodex.RelationshipTypeArtist))

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
	meta := d.dexToMetadata(manga)

	// Preferred language first, then english/romaji/native, then anything
	mangaTitle := manga.GetTitle(d.filter.Language, false)
	if mangaTitle == "" {
		mangaTitle = meta.Title()
	}
	if mangaTitle == "" {
		mangaTitle = manga.GetTitle(d.filter.Language, true)
	}

	var m metadata.Metadata = meta
	return &mango.Manga{
		Title:     mangaTitle,
		URL:       fmt.Sprintf("%stitle/%s", website, manga.ID),
		ID:        manga.ID,
		Cover:     meta.CoverImage,
		Metadata_: &m,
	}
}

func (d *dex) dexToMetadata(manga *mangodex.Manga) *mangadata.Metadata {
	// romaji/native depend on the original language, e.g. "ko-ro"/"ko" for manhwa
	original := manga.Attributes.OriginalLanguage
	englishTitle := manga.GetTitle("en", false)
	romajiTitle := manga.GetTitle(original+"-ro", false)
	nativeTitle := manga.GetTitle(original, false)

	// every other title, in the order MangaDex lists them (case-insensitive duplicates skipped)
	altTitles := []string{}
	seen := map[string]bool{}
	for _, title := range []string{englishTitle, romajiTitle, nativeTitle} {
		seen[strings.ToLower(title)] = true
	}
	for _, titles := range []mangodex.LocalisedStrings{manga.Attributes.Title, manga.Attributes.AltTitles} {
		for _, title := range titles.List {
			key := strings.ToLower(title.Value)
			if !seen[key] {
				seen[key] = true
				altTitles = append(altTitles, title.Value)
			}
		}
	}

	var covers []string
	var authors []string
	var artists []string
	for _, relationship := range manga.Relationships {
		switch relationship.Type {
		case mangodex.RelationshipTypeCoverArt:
			coverRel, _ := relationship.Attributes.(*mangodex.CoverAttributes)
			covers = append(covers, coverRel.FileName)
		case mangodex.RelationshipTypeAuthor:
			authorRel, _ := relationship.Attributes.(*mangodex.AuthorAttributes)
			authors = append(authors, authorRel.Name)
		case mangodex.RelationshipTypeArtist:
			// Same type of attribute as Author
			artistRel, _ := relationship.Attributes.(*mangodex.AuthorAttributes)
			artists = append(artists, artistRel.Name)
		default:
			continue
		}
	}

	var cover string
	if len(covers) != 0 {
		cover = fmt.Sprintf("%scovers/%s/%s", website, manga.ID, covers[0])
	}

	var tags []string
	var genres []string

	for _, tag := range manga.Attributes.Tags {
		name := tag.GetName(d.filter.Language, true)
		// TODO: also group TagGroupContent? need to decide how to separate the tags
		if tag.Attributes.Group == mangodex.TagGroupGenre {
			genres = append(genres, name)
			continue
		}
		tags = append(tags, name)
	}

	var date metadata.Date
	if manga.Attributes.Year != nil {
		// mangadex doesn't provide start month/day
		date = metadata.Date{
			Year:  *manga.Attributes.Year,
			Month: 1,
			Day:   1,
		}
	}

	var status metadata.Status
	if manga.Attributes.Status != nil {
		switch *manga.Attributes.Status {
		case mangodex.PublicationStatusOngoing:
			status = metadata.StatusReleasing
		case mangodex.PublicationStatusCompleted:
			status = metadata.StatusFinished
		case mangodex.PublicationStatusHiatus:
			status = metadata.StatusHiatus
		case mangodex.PublicationStatusCancelled:
			status = metadata.StatusCancelled
		default:
			status = metadata.StatusNotYetReleased // shouldn't happen
		}
	}

	var externalIDs []metadata.ID
	for _, c := range []struct {
		code   metadata.IDCode
		source metadata.IDSource
	}{
		{metadata.IDCodeAnilist, metadata.IDSourceAnilist},
		{metadata.IDCodeMyAnimeList, metadata.IDSourceMyAnimeList},
		{metadata.IDCodeKitsu, metadata.IDSourceKitsu},
		{metadata.IDCodeMangaUpdates, metadata.IDSourceMangaUpdates},
		{metadata.IDCodeAnimePlanet, metadata.IDSourceAnimePlanet},
	} {
		if id := manga.Attributes.Links.GetLocalString(string(c.code), false); id != "" {
			externalIDs = append(externalIDs, metadata.ID{
				Raw:    id,
				Code:   c.code,
				Source: c.source,
			})
		}
	}

	// Mangadex doesn't provide any kind of ID that could be used,
	// so IDProvider is not set but the name is to be able to differentiate
	// when metadata comes from the provider
	return &mangadata.Metadata{
		EnglishTitle:      englishTitle,
		RomajiTitle:       romajiTitle,
		NativeTitle:       nativeTitle,
		Synonyms:          altTitles,
		Summary:           manga.GetDescription(d.filter.Language, true),
		CoverImage:        cover,
		TagList:           tags,
		GenreList:         genres,
		AuthorList:        authors,
		ArtistList:        artists,
		DateStart:         date,
		PublicationStatus: status,
		SourceURL:         fmt.Sprintf("%stitle/%s", website, manga.ID),
		ProviderIDCode:    "dex",
		OtherIDs:          externalIDs,
	}
}
