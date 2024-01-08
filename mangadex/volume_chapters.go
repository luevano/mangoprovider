package mangadex

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangodex"
	"github.com/luevano/mangoprovider/mango"
	"github.com/philippgille/gokv"
)

func (d *Dex) VolumeChapters(ctx context.Context, store gokv.Store, volume mango.MangoVolume) ([]libmangal.Chapter, error) {
	var chapters []libmangal.Chapter

	language := d.options.Language
	// TODO: use incoming options instead of checking for empty
	if language == "" {
		language = "en"
	}

	// mangadex api returns "none" for "non-volumed" chapters,
	// which are saved as 0 in libmangal.Volume
	volumeNumber := "none"
	if volume.Number != 0 {
		volumeNumber = volume.String()
	}

	// TODO: add scanlation group once libmangal is modified to accept it
	params := url.Values{}
	params.Set("manga", volume.Manga().Info().ID)
	params.Set("volume[]", volumeNumber)
	params.Set("limit", strconv.Itoa(100))
	params.Set("order[chapter]", mangodex.OrderAscending)
	params.Set("translatedLanguage[]", language)

	ratings := []mangodex.ContentRating{mangodex.ContentRatingSafe, mangodex.ContentRatingSuggestive}
	if d.options.NSFW {
		ratings = append(ratings, mangodex.ContentRatingPorn)
		ratings = append(ratings, mangodex.ContentRatingErotica)
	}
	for _, rating := range ratings {
		params.Add("contentRating[]", string(rating))
	}

	// need a query with params included for the cache,
	// this doesn't include the offset so that it retrieves and saves all the chapters
	queryWithParams := params.Encode()

	found, err := store.Get(queryWithParams, &chapters)
	if err != nil {
		return nil, err
	}
	if found {
		// TODO: use logger
		// fmt.Printf("found mangas in cache with query %q\n", query)
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

	err = store.Set(queryWithParams, chapters)
	if err != nil {
		return nil, err
	}

	return chapters, nil
}

func (d *Dex) populateChapters(store gokv.Store, offset int, params url.Values, volume mango.MangoVolume) ([]libmangal.Chapter, bool, error) {
	var chapters []libmangal.Chapter

	volumeNumber := params.Get("volume[]")
	params.Set("offset", strconv.Itoa(offset))

	chapterList, err := d.client.Chapter.List(params)
	if err != nil {
		// TODO: need to start using the logger, need to receive a logger, check options
		// log.Fatalln(err)
		return nil, false, err
	}

	if len(chapterList) == 0 {
		return nil, true, nil
	}

	for _, chapter := range chapterList {
		// Skip external chapters (can't be downloaded)
		// TODO: add option to accept unavailable (external url) chapters?
		if chapter.Attributes.ExternalURL != nil {
			continue
		}

		chapterTitle := chapter.GetTitle()
		chapterID := chapter.ID
		chapterNumberStr := chapter.GetChapterNum()

		if chapterNumberStr == "-" {
			return nil, false, fmt.Errorf("chapter number for manga %q volume %q with title %q wasn't found", volume.Manga_.Info().Title, volumeNumber, chapterTitle)
		}

		chapterNumber, err := strconv.ParseFloat(chapterNumberStr, 64)
		if err != nil {
			return nil, false, err
		}

		if chapterTitle == "" {
			chapterTitle = fmt.Sprintf("Chapter %06.1f", chapterNumber)
		}

		c := mango.MangoChapter{
			Title:   chapterTitle,
			URL:     fmt.Sprintf("https://mangadex.org/chapter/%s", chapterID),
			Number:  float32(chapterNumber),
			Volume_: &volume,
		}

		chapters = append(chapters, c)
	}

	// if received 100 entries, that means it probably has more
	if len(chapterList) == 100 {
		return chapters, false, nil
	}

	return chapters, true, nil
}
