package mangaplus

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/luevano/libmangal"
	"github.com/luevano/mangoplus"
	mango "github.com/luevano/mangoprovider"
	"github.com/philippgille/gokv"
)

// TODO: handle case with 0 chapters?

func (p *plus) VolumeChapters(ctx context.Context, store gokv.Store, volume mango.Volume) ([]libmangal.Chapter, error) {
	var chapters []libmangal.Chapter

	cacheID := volume.Manga_.String()
	found, err := store.Get(cacheID, &chapters)
	if err != nil {
		return nil, err
	}
	if found {
		mango.Log(fmt.Sprintf("Found chapters in cache for volume %s", volume.String()))
		return chapters, nil
	}

	mangaDetails, err := p.client.Manga.Get(volume.Manga_.ID)
	if err != nil {
		return nil, err
	}
	chapterListGroup := mangaDetails.ChapterListGroup

	lastMainChapNum := float32(0.0)
	for _, chapterGroup := range chapterListGroup {
		var chapterLists []mangoplus.Chapter
		chapterLists = append(chapterLists, chapterGroup.FirstChapterList...)
		chapterLists = append(chapterLists, chapterGroup.LastChapterList...)

		for _, chapter := range chapterLists {
			// Initialize to -1.0 to keep track of failed to parse numbers
			currChapNum := float32(-1.0)
			title := chapter.Name
			chNumMatch := mango.ChapterNumberRegex.FindString(title)
			if chNumMatch != "" {
				number, err := strconv.ParseFloat(chNumMatch, 32)
				if err == nil {
					currChapNum = float32(number)
				}
			}
			if chapter.SubTitle != nil {
				title = *chapter.SubTitle
			}

			// TODO: enhance these checks, need to test it further
			//
			// When the title is explicitly a "Bonus" or "Extra",
			// and if the current chapter number wasn't parsed
			if (fuzzy.MatchNormalizedFold("bonus", title) ||
				fuzzy.MatchNormalizedFold("ex", title)) &&
				currChapNum == float32(-1.0) {
				// What if there are 2 bonus chapters back to back?
				// Need to add 0.1 instead I guess...
				currChapNum = lastMainChapNum + float32(0.5)
			}

			chNameMatch := mango.ChapterNameRegex.FindStringSubmatch(title)
			if len(chNameMatch) > 2 {
				title = strings.TrimSpace(chNameMatch[2])
			}

			timeStamp := time.Unix(int64(chapter.StartTimeStamp), 0)
			date := libmangal.Date{
				Year:  timeStamp.Year(),
				Month: int(timeStamp.Month()),
				Day:   timeStamp.Day(),
			}

			c := mango.Chapter{
				Title:           title,
				ID:              strconv.Itoa(chapter.ChapterId),
				URL:             fmt.Sprintf("%sviewer/%d", website, chapter.ChapterId),
				Number:          currChapNum,
				Date:            date,
				ScanlationGroup: "MangaPlus",
				Volume_:         &volume,
			}
			chapters = append(chapters, &c)

			// Keep track of the latest "main" (integer) chapter number
			if currChapNum == float32(int(currChapNum)) {
				lastMainChapNum = currChapNum
			}
		}
	}

	return chapters, nil
}
