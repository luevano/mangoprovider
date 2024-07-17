package mangadex

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/mangodex"
	mango "github.com/luevano/mangoprovider"
)

// Contains the actual chapter list as well as helper values for filtering.
type aggregate struct {
	chapters    []mangadata.Chapter
	chaptersMap map[string][]mangadata.Chapter
	groupsCount map[string]int
}

func (d *dex) VolumeChapters(ctx context.Context, store mango.Store, volume mango.Volume) ([]mangadata.Chapter, error) {
	limit := 100
	agg := aggregate{
		chapters:    []mangadata.Chapter{},
		chaptersMap: map[string][]mangadata.Chapter{},
		groupsCount: map[string]int{},
	}

	// Mangadex api returns "none" for "non-volumed" chapters,
	// which are saved as -1.0 in libmangal.Volume
	volumeNumber := "none"
	if !volume.None {
		volumeNumber = volume.String()
	}

	params := url.Values{}
	params.Set("manga", volume.Manga_.ID)
	params.Add("volume[]", volumeNumber)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("order[chapter]", mangodex.OrderAscending)
	params.Add("translatedLanguage[]", d.filter.Language)
	params.Add("includes[]", string(mangodex.RelationshipTypeScanlationGroup))
	params.Add("includes[]", string(mangodex.RelationshipTypeUser))

	ratings := []mangodex.ContentRating{mangodex.ContentRatingSafe, mangodex.ContentRatingSuggestive}
	if d.filter.NSFW {
		ratings = append(ratings, mangodex.ContentRatingPorn)
		ratings = append(ratings, mangodex.ContentRatingErotica)
	}
	for _, rating := range ratings {
		params.Add("contentRating[]", string(rating))
	}

	// This doesn't include the offset so that it retrieves and saves all the chapters.
	cacheID := fmt.Sprintf("%s-%s", params.Encode(), d.filter.String())
	var chapters []mangadata.Chapter
	found, err := store.GetChapters(cacheID, volume, &chapters)
	if err != nil {
		return nil, err
	}
	if found {
		return chapters, nil
	}

	offset := 0
	for {
		// The offset is set on each iteration, shouldn't be included in the cacheID.
		params.Set("offset", strconv.Itoa(offset))
		ended, err := d.populateChapters(&agg, params, volume)
		if err != nil {
			return nil, err
		}
		if ended {
			break
		}
		offset += limit
	}

	// TODO: add option to exclude list of scanlators/prefer list of scanlators
	var chaptersFiltered []mangadata.Chapter

	if d.filter.AvoidDuplicateChapters {
		chaptersFiltered, err = getUniqueChapters(&agg)
		if err != nil {
			return nil, err
		}
	} else {
		chaptersFiltered = agg.chapters
	}

	sort.SliceStable(chaptersFiltered, func(i, j int) bool {
		return chaptersFiltered[i].Info().Number < chaptersFiltered[j].Info().Number
	})

	err = store.SetChapters(cacheID, chaptersFiltered)
	if err != nil {
		return nil, err
	}

	return chaptersFiltered, nil
}

// Make the request and parse the responses, populating the actual chapter list and extra info useful for filtering.
func (d *dex) populateChapters(agg *aggregate, params url.Values, volume mango.Volume) (bool, error) {
	chapterList, err := d.client.Chapter.List(params)
	if err != nil {
		return false, err
	}

	if len(chapterList) == 0 {
		return true, nil
	}

	for _, chapter := range chapterList {
		// Skip external chapters (can't be downloaded) unless wanted.
		if chapter.Attributes.ExternalURL != nil && !d.filter.ShowUnavailableChapters {
			continue
		}

		var chapterTitle string
		if chapter.GetTitle() != "" {
			chapterTitle = chapter.GetTitle()
		}

		chapterNumber, chapterTitleNumber, err := getChapterNum(chapter)
		if err != nil {
			return false, err
		}
		// Add "Chapter #" when no title for the chapter is found.
		if chapterTitle == "" {
			if chapterTitle == "" {
				chapterTitle = chapterTitleNumber
			} else {
				chapterTitle = fmt.Sprintf("%s - %s", chapterTitleNumber, chapterTitle)
			}
		}

		chapterID := chapter.ID
		date := getDate(chapter.Attributes.PublishAt)
		scanlator := getScanlator(chapter.Relationships)

		c := mango.Chapter{
			Title:           chapterTitle,
			ID:              chapterID,
			URL:             fmt.Sprintf("%schapter/%s", website, chapterID),
			Number:          chapterNumber,
			Date:            date,
			ScanlationGroup: scanlator,
			Volume_:         &volume,
		}

		mapKey := strings.Split(chapterTitleNumber, " ")[1]
		agg.chaptersMap[mapKey] = append(agg.chaptersMap[mapKey], &c)
		agg.groupsCount[scanlator] += 1
		agg.chapters = append(agg.chapters, &c)
	}

	// If received 100 entries means it probably has more.
	if len(chapterList) == 100 {
		return false, nil
	}

	return true, nil
}
