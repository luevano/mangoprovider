package mangadex

import (
	"fmt"
	"strconv"
	"time"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	"github.com/luevano/mangodex"
	mango "github.com/luevano/mangoprovider"
)

// TODO: make this configurable?
const chapterNumberFormat = "Chapter %s"

// getChapterNum: The chapter's number, false for chapters without one (oneshots, extras).
func getChapterNum(chapter *mangodex.Chapter) (float32, bool, error) {
	chapterNumberStr := chapter.GetChapterNum()
	if chapterNumberStr == "-" {
		return 0, false, nil
	}

	chapterNumber, err := strconv.ParseFloat(chapterNumberStr, 32)
	if err != nil {
		return 0, false, err
	}
	return float32(chapterNumber), true, nil
}

// numberExtraChapters: Chapters without a number go right after the previous one (in the api's
// order), untitled ones are named after their number.
func numberExtraChapters(chapters []mangadata.Chapter, extras map[*mango.Chapter]bool) {
	var last float32
	for _, chapter := range chapters {
		c := chapter.(*mango.Chapter)
		if extras[c] {
			c.Number = mango.NextExtraNumber(last)
		}
		if c.Title == "" {
			c.Title = fmt.Sprintf(chapterNumberFormat, mango.FormattedFloat(c.Number))
		}
		last = c.Number
	}
}

// Get parsed published date or just today's date.
func (d *dex) getDate(publishAt string) metadata.Date {
	publishedDate, err := time.Parse(time.RFC3339, publishAt)
	if err != nil {
		d.log.Log("failed to parse chapter date, using today")
		now := time.Now()
		return mango.DateOf(now)

	}
	return mango.DateOf(publishedDate)
}

// Finds the first scanlator or the first username or "mangadex".
func (d *dex) getScanlator(relationships []*mangodex.Relationship) string {
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
		d.log.Log("no scanlator for chapter, using username")
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
		scanlator = "MangaDex"
		d.log.Log("no scanlator or username for chapter, defaulting to %q", scanlator)
	}
	return scanlator
}

// Filters out duplicate chapters, the logic tries to get the chapter that's from a scanlator that appears the most in the volume.
func getUniqueChapters(agg *aggregate) ([]mangadata.Chapter, error) {
	var chaptersFiltered []mangadata.Chapter
	for number, chapters := range agg.chaptersMap {
		switch len(chapters) {
		case 0:
			return nil, fmt.Errorf("remove duplicate chapters: no chapters for number %q", number)
		case 1:
			chaptersFiltered = append(chaptersFiltered, chapters[0])
		default:
			// prefer a downloadable upload over an unavailable one
			candidates := chapters
			var available []mangadata.Chapter
			for _, chapter := range chapters {
				if chapter.Info().Unavailable == "" {
					available = append(available, chapter)
				}
			}
			if len(available) != 0 {
				candidates = available
			}

			var chapterTemp mangadata.Chapter
			maxUploads := 0
			for _, chapter := range candidates {
				scanlator := chapter.Info().ScanlationGroup
				// uploads is the total times the scanlator appears in the volume (how many uploads)
				uploads, found := agg.groupsCount[scanlator]
				if !found {
					return nil, fmt.Errorf("remove duplicate chapters: uploads of group %q not counted", scanlator)
				}
				// if it has appeared the most then select it as a candidate for the chapter #
				if uploads > maxUploads {
					chapterTemp = chapter
					maxUploads = uploads
				}
			}
			if chapterTemp == nil {
				return nil, fmt.Errorf("remove duplicate chapters: no candidate for number %q", number)
			}

			chaptersFiltered = append(chaptersFiltered, chapterTemp)
		}
	}
	return chaptersFiltered, nil
}
