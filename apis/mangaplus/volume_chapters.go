package mangaplus

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
)

func (p *plus) VolumeChapters(ctx context.Context, store mango.Store, volume mango.Volume) ([]mangadata.Chapter, error) {
	var chapters []mangadata.Chapter

	mangaID := volume.Manga_.ID
	found, err := store.GetChapters(mangaID, volume, &chapters)
	if err != nil {
		return nil, err
	}
	if found {
		return chapters, nil
	}

	err = p.searchChapters(ctx, &chapters, volume, mangaID)
	if err != nil {
		return nil, err
	}

	err = store.SetChapters(mangaID, chapters)
	if err != nil {
		return nil, err
	}

	return chapters, nil
}

func (p *plus) searchChapters(ctx context.Context, chapters *[]mangadata.Chapter, volume mango.Volume, id string) error {
	mangaDetails, err := p.client.Manga.GetContext(ctx, id)
	if err != nil {
		return err
	}
	chapterList := mangaDetails.ChapterListV2

	// All chapters are assumed to come in order, there is no other way to deal
	// with extra/bonus chapters (if they don't come with a number)
	lastNumber := float32(0.0)
	for _, chapter := range chapterList {
		title := chapter.Name
		if chapter.SubTitle != "" {
			title = mango.ParseChapterTitle(chapter.SubTitle)
		}
		c := mango.Chapter{
			Title:           title,
			ID:              strconv.Itoa(int(chapter.ChapterId)),
			URL:             fmt.Sprintf("%sviewer/%d", website, chapter.ChapterId),
			Number:          parseChapterNumber(chapter.Name, lastNumber),
			Date:            parseTSSecs(chapter.StartTimeStamp),
			ScanlationGroup: "MangaPlus",
			Volume_:         &volume,
		}
		*chapters = append(*chapters, &c)
		lastNumber = c.Number
	}
	return nil
}

func parseChapterNumber(s string, lastNumber float32) float32 {
	number, ok := mango.ParseChapterNumber(s)
	// extras have no number, or repeat the last one
	if !ok || number == lastNumber {
		return mango.NextExtraNumber(lastNumber)
	}
	return number
}

func parseTSSecs(timestamp int64) metadata.Date {
	ts := time.Unix(timestamp, 0)
	return metadata.Date{
		Year:  ts.Year(),
		Month: int(ts.Month()),
		Day:   ts.Day(),
	}
}
