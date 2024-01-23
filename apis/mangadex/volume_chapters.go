package mangadex

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangodex"
	mango "github.com/luevano/mangoprovider"
	"github.com/philippgille/gokv"
)

func (d *dex) VolumeChapters(ctx context.Context, store gokv.Store, volume mango.Volume) ([]libmangal.Chapter, error) {
	chapters := []libmangal.Chapter{}

	// Mangadex api returns "none" for "non-volumed" chapters,
	// which are saved as 0 in libmangal.Volume
	volumeNumber := "none"
	if volume.Number != 0 {
		volumeNumber = volume.String()
	}

	params := url.Values{}
	params.Set("manga", volume.Manga_.ID)
	params.Add("volume[]", volumeNumber)
	params.Set("limit", strconv.Itoa(100))
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

	// This doesn't include the offset so that it retrieves and saves all the chapters
	cacheID := fmt.Sprintf("%s-%s", params.Encode(), d.filter.String())

	found, err := store.Get(cacheID, &chapters)
	if err != nil {
		return nil, err
	}
	if found {
		mango.Log(fmt.Sprintf("[%s]found chapters in cache for manga %q with id %q", providerInfo.ID, volume.Manga_.Title, volume.Manga_.ID))
		return chapters, nil
	}

	offset := 0
	for {
		ended, err := d.populateChapters(&chapters, offset, params, volume)
		if err != nil {
			return nil, err
		}
		if ended {
			break
		}
		offset += 100
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

func (d *dex) populateChapters(chapters *[]libmangal.Chapter, offset int, params url.Values, volume mango.Volume) (bool, error) {
	params.Set("offset", strconv.Itoa(offset))
	chapterList, err := d.client.Chapter.List(params)
	if err != nil {
		return false, err
	}

	if len(chapterList) == 0 {
		return true, nil
	}

	for _, chapter := range chapterList {
		// Skip external chapters (can't be downloaded) unless wanted
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
		// Add "Chapter #" when wanted or when no title for the chapter is found.
		if d.filter.TitleChapterNumber || chapterTitle == "" {
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
			URL:             fmt.Sprintf("https://mangadex.org/chapter/%s", chapterID),
			Number:          chapterNumber,
			Date:            date,
			ScanlationGroup: scanlator,
			Volume_:         &volume,
		}

		*chapters = append(*chapters, c)
	}

	// If received 100 entries means it probably has more
	if len(chapterList) == 100 {
		return false, nil
	}

	return true, nil
}

func getChapterNum(chapter *mangodex.Chapter) (float32, string, error) {
	chapterNumberStr := chapter.GetChapterNum()
	chapterNumber, err := strconv.ParseFloat(chapterNumberStr, 32)
	if err != nil {
		return 0.0, "", err
	}

	chapterTitleNumber := fmt.Sprintf("Chapter %06.1f", chapterNumber)
	return float32(chapterNumber), chapterTitleNumber, nil
}

func getDate(publishAt string) libmangal.Date {
	publishedDate, err := time.Parse(time.RFC3339, publishAt)
	if err != nil {
		mango.Log("failed to parse chapter date, using today")
		now := time.Now()
		return libmangal.Date{
			Year:  now.Year(),
			Month: int(now.Month()),
			Day:   now.Day(),
		}

	}
	return libmangal.Date{
		Year:  publishedDate.Year(),
		Month: int(publishedDate.Month()),
		Day:   publishedDate.Day(),
	}
}

func getScanlator(relationships []*mangodex.Relationship) string {
	var scanlator string
	for _, relationship := range relationships {
		if relationship.Type == mangodex.RelationshipTypeScanlationGroup {
			groupRel, _ := relationship.Attributes.(*mangodex.ScanlationGroupAttributes)
			scanlator = groupRel.Name
			break
		}
	}
	// If no scanlator group is linked to the chapter, use the uploader user
	if scanlator == "" {
		for _, relationship := range relationships {
			if relationship.Type == mangodex.RelationshipTypeUser {
				userRel, _ := relationship.Attributes.(*mangodex.UserAttributes)
				scanlator = userRel.Username
				break
			}
		}
	}
	// If even then the scanlator is not set, just use "mangadex"
	if scanlator == "" {
		scanlator = "mangadex"
	}
	return scanlator
}
