package mangaplus

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	"github.com/luevano/mangoplus"
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

	err = p.searchChapters(&chapters, volume, mangaID)
	if err != nil {
		return nil, err
	}

	err = store.SetChapters(mangaID, chapters)
	if err != nil {
		return nil, err
	}

	return chapters, nil
}

func (p *plus) searchChapters(chapters *[]mangadata.Chapter, volume mango.Volume, id string) error {
	mangaDetails, err := p.client.Manga.Get(id)
	if err != nil {
		return err
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
			c := mango.Chapter{
				Title:           parseChapterTitle(chapter.Name, chapter.SubTitle),
				ID:              strconv.Itoa(chapter.ChapterId),
				URL:             fmt.Sprintf("%sviewer/%d", website, chapter.ChapterId),
				Number:          parseChapterNumber(chapter.Name, lastNumber),
				Date:            parseTSSecs(chapter.StartTimeStamp),
				ScanlationGroup: "MangaPlus",
				Volume_:         &volume,
			}
			*chapters = append(*chapters, &c)
			lastNumber = c.Number
		}
	}
	return nil
}

func parseChapterNumber(s string, lastNumber float32) float32 {
	number := float32(-1.0)
	chNumMatch := mango.ChapterNumberRegex.FindString(s)
	if chNumMatch != "" {
		// Special case fo MangaPlus as it's "decimal" numbers contain "-"
		chNumMatch = strings.Replace(chNumMatch, "-", ".", 1)
		number64, err := strconv.ParseFloat(chNumMatch, 32)
		if err == nil {
			number = float32(number64)
		}
	}
	// If either there was no match for the number or
	// parsing the number failed for some reason, or if the number is the same as the last
	if number == float32(-1.0) || number == lastNumber {
		// If it's the first extra, make it 0.5, else add 0.1
		// Using a trick to avoid floating point precision issues https://stackoverflow.com/a/56300186
		if mango.FloatIsInt(lastNumber) {
			number = float32((float64(lastNumber*10.0) + float64(5.0)) / 10.0)
		} else {
			number = float32((float64(lastNumber*10.0) + float64(1.0)) / 10.0)
		}
	}
	return number
}

func parseChapterTitle(s string, subTitle *string) string {
	title := s
	if subTitle != nil {
		// Need to normalize the spaces, some weird unicode spaces are not matched with regex
		title = strings.TrimSpace(strings.Join(strings.Fields(strings.Replace(*subTitle, "\t", " ", -1)), " "))

		// Try to get the name without prefix "Chapter 123:" or similar
		matchGroups := mango.ReNamedGroups(mango.ChapterNameRegex, title)
		titleTemp := strings.TrimSpace(matchGroups[mango.ChapterNameIDName])
		if titleTemp != "" {
			// Check that the resulting title is not just "Part 123",
			// as it probably is part of the whole title and we'll like to keep
			// the prefix
			// This happens with Spy x Family: "Mission X Part Y" for example
			if !mango.ChapterNameExcludeRegex.MatchString(titleTemp) {
				title = titleTemp
				partNum := strings.TrimSpace(matchGroups[mango.ChapterPartNumberIDName])
				if partNum != "" {
					title = fmt.Sprintf("%s, Part %s", title, partNum)
				}
			}
		}
	}
	return title
}

func parseTSSecs(timestamp int) metadata.Date {
	ts := time.Unix(int64(timestamp), 0)
	return metadata.Date{
		Year:  ts.Year(),
		Month: int(ts.Month()),
		Day:   ts.Day(),
	}
}
