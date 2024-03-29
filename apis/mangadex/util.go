package mangadex

import (
	"fmt"
	"strconv"
	"time"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangodex"
	mango "github.com/luevano/mangoprovider"
)

// Get parsed chapter number as a float and string.
func getChapterNum(chapter *mangodex.Chapter) (float32, string, error) {
	chapterNumberStr := chapter.GetChapterNum()
	chapterNumber, err := strconv.ParseFloat(chapterNumberStr, 32)
	if err != nil {
		return 0.0, "", err
	}

	chapterTitleNumber := fmt.Sprintf(fmt.Sprintf("Chapter %s", chapterNumberFormat), chapterNumber)
	return float32(chapterNumber), chapterTitleNumber, nil
}

// Get parsed published date or just today's date.
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

// Finds the first scanlator or the first username or "mangadex".
func getScanlator(relationships []*mangodex.Relationship) string {
	var scanlator string
	for _, relationship := range relationships {
		if relationship.Type == mangodex.RelationshipTypeScanlationGroup {
			groupRel, _ := relationship.Attributes.(*mangodex.ScanlationGroupAttributes)
			scanlator = groupRel.Name
			break
		}
	}
	// If no scanlator group is linked to the chapter, use the uploader user.
	if scanlator == "" {
		mango.Log("no scanlator for chapter, using username")
		for _, relationship := range relationships {
			if relationship.Type == mangodex.RelationshipTypeUser {
				userRel, _ := relationship.Attributes.(*mangodex.UserAttributes)
				scanlator = userRel.Username
				break
			}
		}
	}
	// If even then the scanlator is not set, just use "mangadex".
	if scanlator == "" {
		mango.Log("no scanlator or username for chapter, defaulting to 'mangadex'")
		scanlator = "MangaDex"
	}
	return scanlator
}

// Filters out duplicate chapters, the logic tries to get the chapter that's from a scanlator that appears the most in the volume.
func getUniqueChapters(agg *aggregate) ([]libmangal.Chapter, error) {
	var chaptersFiltered []libmangal.Chapter
	for _, chapters := range agg.chaptersMap {
		switch len(chapters) {
		case 0:
			return nil, fmt.Errorf("unexpected error; len(chapters) == 0 at AvoidDuplicateChapters")
		case 1:
			chaptersFiltered = append(chaptersFiltered, chapters[0])
		default:
			var chapterTemp libmangal.Chapter
			maxUploads := 0
			for _, chapter := range chapters {
				scanlator := chapter.Info().ScanlationGroup
				// uploads is the total times the scanlator appears in the volume (how many uploads)
				uploads, found := agg.groupsCount[scanlator]
				if !found {
					return nil, fmt.Errorf("unexpected error; groupsCount[scanlator] not found at AvoidDuplicateChapters")
				}
				// if it has appeared the most then select it as a candidate for the chapter #
				if uploads > maxUploads {
					chapterTemp = chapter
					maxUploads = uploads
				}
			}
			if chapterTemp == nil {
				return nil, fmt.Errorf("unexpected error; chapterTemp == nil at AvoidDuplicateChapters")
			}

			chaptersFiltered = append(chaptersFiltered, chapterTemp)
		}
	}
	return chaptersFiltered, nil
}
