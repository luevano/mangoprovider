package mangaplus

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangoplus"
	mango "github.com/luevano/mangoprovider"
	"github.com/philippgille/gokv"
)

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

	// All chapters are assumed to come in order, there is no other way to deal
	// with extra/bonus chapters (if they don't come with a number)
	lastNumber := float32(0.0)
	for _, chapterGroup := range chapterListGroup {
		var chapterLists []mangoplus.Chapter
		chapterLists = append(chapterLists, chapterGroup.FirstChapterList...)
		chapterLists = append(chapterLists, chapterGroup.MidChapterList...)
		chapterLists = append(chapterLists, chapterGroup.LastChapterList...)

		for _, chapter := range chapterLists {
			number := float32(-1.0)
			title := chapter.Name
			chNumMatch := mango.ChapterNumberRegex.FindString(title)
			if chNumMatch != "" {
				number64, err := strconv.ParseFloat(chNumMatch, 32)
				if err == nil {
					number = float32(number64)
				}
			}
			// If either there was no match for the number or
			// parsing the number failed for some reason
			if number == float32(-1.0) {
				// If it's the first extra, make it 0.5, else add 0.1
				if mango.FloatIsInt(lastNumber) {
					number = lastNumber + float32(0.5)
				} else {
					number = lastNumber + float32(0.1)
				}
			}
			lastNumber = number

			if chapter.SubTitle != nil {
				title = *chapter.SubTitle
			}

			// Try to get the name without prefix "Chapter 123:" or similar
			matchGroups := mango.ReNamedGroups(mango.ChapterNameRegex, title)
			titleTemp, found := matchGroups[mango.ChapterNameIDName]
			if found {
				title = titleTemp
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
				Number:          number,
				Date:            date,
				ScanlationGroup: "MangaPlus",
				Volume_:         &volume,
			}
			chapters = append(chapters, &c)
		}
	}
	return chapters, nil
}
