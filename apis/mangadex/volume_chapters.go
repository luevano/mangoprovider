package mangadex

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangodex"
	"github.com/luevano/mangoprovider/mango"
	"github.com/philippgille/gokv"
)

func (d *dex) VolumeChapters(ctx context.Context, logger *libmangal.Logger, store gokv.Store, volume mango.Volume) ([]libmangal.Chapter, error) {
	var chapters []libmangal.Chapter

	// Mangadex api returns "none" for "non-volumed" chapters,
	// which are saved as 0 in libmangal.Volume
	volumeNumber := "none"
	if volume.Number != 0 {
		volumeNumber = volume.String()
	}

	params := url.Values{}
	params.Set("manga", volume.Manga_.ID)
	params.Set("volume[]", volumeNumber)
	params.Set("limit", strconv.Itoa(100))
	params.Set("order[chapter]", mangodex.OrderAscending)
	params.Set("translatedLanguage[]", d.filter.Language)
	params.Set("includes[]", string(mangodex.RelationshipTypeScanlationGroup))

	ratings := []mangodex.ContentRating{mangodex.ContentRatingSafe, mangodex.ContentRatingSuggestive}
	if d.filter.NSFW {
		ratings = append(ratings, mangodex.ContentRatingPorn)
		ratings = append(ratings, mangodex.ContentRatingErotica)
	}
	for _, rating := range ratings {
		params.Add("contentRating[]", string(rating))
	}

	// This doesn't include the offset so that it retrieves and saves all the chapters
	cacheID := fmt.Sprintf("%s-%s", params.Encode(), d.filter.String())

	found, err := store.Get(cacheID, &chapters)
	if err != nil {
		return nil, err
	}
	if found {
		logger.Log(fmt.Sprintf("[%s]found chapters in cache for manga %q with id %q", providerInfo.ID, volume.Manga_.Title, volume.Manga_.ID))
		return chapters, nil
	}

	offset := 0
	for {
		chaptersTemp, ended, err := d.populateChapters(store, offset, params, volume)
		if err != nil {
			return nil, err
		}
		offset += 100

		if chaptersTemp != nil {
			chapters = append(chapters, chaptersTemp...)
		}

		if ended {
			break
		}
	}

	// TODO: add option to exclude list of scanlators/prefer list of scanlators
	var chaptersFiltered []libmangal.Chapter

	if d.filter.AvoidDuplicateChapters {
		allFound := map[float32]bool{}
		for _, chapter := range chapters {
			_, found := allFound[chapter.Info().Number]
			if !found {
				chaptersFiltered = append(chaptersFiltered, chapter)
				allFound[chapter.Info().Number] = true
			}
		}
	} else {
		chaptersFiltered = chapters
	}

	err = store.Set(cacheID, chaptersFiltered)
	if err != nil {
		return nil, err
	}

	return chaptersFiltered, nil
}

func (d *dex) populateChapters(store gokv.Store, offset int, params url.Values, volume mango.Volume) ([]libmangal.Chapter, bool, error) {
	var chapters []libmangal.Chapter

	volumeNumber := params.Get("volume[]")
	params.Set("offset", strconv.Itoa(offset))

	chapterList, err := d.client.Chapter.List(params)
	if err != nil {
		return nil, false, err
	}

	if len(chapterList) == 0 {
		return nil, true, nil
	}

	for _, chapter := range chapterList {
		// Skip external chapters (can't be downloaded) unless wanted
		if chapter.Attributes.ExternalURL != nil && !d.filter.ShowUnavailableChapters {
			continue
		}

		chapterTitleRaw := chapter.GetTitle()
		chapterID := chapter.ID
		chapterNumberStr := chapter.GetChapterNum()

		if chapterNumberStr == "-" {
			return nil, false, fmt.Errorf("chapter number for manga %q volume %q with title %q wasn't found", volume.Manga_.Title, volumeNumber, chapterTitleRaw)
		}

		chapterNumber, err := strconv.ParseFloat(chapterNumberStr, 32)
		if err != nil {
			return nil, false, err
		}

		// Add "Chapter #" when wanted or when no title for the chapter is found.
		var chapterTitle string
		if chapterTitleRaw != "" {
			chapterTitle = chapterTitleRaw
		}
		chapterTitleNumber := fmt.Sprintf("Chapter %06.1f", chapterNumber)
		if d.filter.TitleChapterNumber || chapterTitle == "" {
			if chapterTitle == "" {
				chapterTitle = chapterTitleNumber
			} else {
				chapterTitle = fmt.Sprintf("%s - %s", chapterTitleNumber, chapterTitle)
			}
		}

		var chapterDate libmangal.Date
		date, err := time.Parse(time.RFC3339, chapter.Attributes.PublishAt)
		if err == nil {
			chapterDate.Year = date.Year()
			chapterDate.Month = int(date.Month())
			chapterDate.Day = date.Day()
		}

		var scanlator string
		for _, relationship := range chapter.Relationships {
			if relationship.Type == mangodex.RelationshipTypeScanlationGroup {
				groupRel, ok := relationship.Attributes.(*mangodex.ScanlationGroupAttributes)
				if !ok {
					return nil, false, fmt.Errorf("unexpected error, failed to convert relationship attribute to scanlation_group type despite being of type %q", mangodex.RelationshipTypeScanlationGroup)
				}

				scanlator = groupRel.Name
				break
			}
		}

		c := mango.Chapter{
			Title:           chapterTitle,
			ID:              chapterID,
			URL:             fmt.Sprintf("https://mangadex.org/chapter/%s", chapterID),
			Number:          float32(chapterNumber),
			Date:            chapterDate,
			ScanlationGroup: scanlator,
			Volume_:         &volume,
		}

		chapters = append(chapters, c)
	}

	// If received 100 entries means it probably has more
	if len(chapterList) == 100 {
		return chapters, false, nil
	}

	return chapters, true, nil
}
