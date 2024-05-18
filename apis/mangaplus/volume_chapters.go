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

// TODO: handle case with 0 chapters

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

	// TODO: handle chapter numbers correctly
	tempChapNum := float32(1.0)
	for _, chapterGroup := range chapterListGroup {
		var chapterLists []mangoplus.Chapter
		chapterLists = append(chapterLists, chapterGroup.FirstChapterList...)
		chapterLists = append(chapterLists, chapterGroup.LastChapterList...)

		for _, chapter := range chapterLists {
			// TODO: use helper methods once implemented
			title := chapter.Name
			if chapter.SubTitle != nil {
				title = *chapter.SubTitle
			}

			timeStamp := time.Unix(int64(chapter.StartTimeStamp), 0)
			date := libmangal.Date{
				Year: timeStamp.Year(),
				Month: int(timeStamp.Month()),
				Day: timeStamp.Day(),
			}

			c := mango.Chapter{
				Title:           title,
				ID:              strconv.Itoa(chapter.ChapterId),
				URL:             fmt.Sprintf("%sviewer/%d", website, chapter.ChapterId),
				Number:          tempChapNum,
				Date:            date,
				ScanlationGroup: "MangaPlus",
				Volume_:         &volume,
			}
			chapters = append(chapters, &c)
			tempChapNum += float32(1.0)
		}
	}

	return chapters, nil
}
