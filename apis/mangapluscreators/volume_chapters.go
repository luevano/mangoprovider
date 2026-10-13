package mangapluscreators

import (
	"context"
	"fmt"
	"time"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
)

func (c *mpc) VolumeChapters(ctx context.Context, store mango.Store, volume mango.Volume) ([]mangadata.Chapter, error) {
	var chapters []mangadata.Chapter

	mangaID := volume.Manga_.ID
	found, err := store.GetChapters(mangaID, volume, &chapters)
	if err != nil {
		return nil, err
	}
	if found {
		return chapters, nil
	}

	err = c.searchChapters(ctx, &chapters, volume)
	if err != nil {
		return nil, err
	}

	err = store.SetChapters(mangaID, chapters)
	if err != nil {
		return nil, err
	}

	return chapters, nil
}

func (c *mpc) searchChapters(ctx context.Context, chapters *[]mangadata.Chapter, volume mango.Volume) error {
	page := 1
	for {
		chaptersDTO, err := c.client.Chapter.ListContext(ctx, volume.Manga_.ID, page)
		if err != nil {
			return err
		}

		if chaptersDTO.EpisodeList == nil || len(*chaptersDTO.EpisodeList) == 0 {
			break
		}
		// All chapters are assumed to come in order, there is no other way to deal
		// with extra/bonus chapters (if they don't come with a number)
		for _, chapter := range *chaptersDTO.EpisodeList {
			c := mango.Chapter{
				Title:           chapter.EpisodeTitle,
				ID:              chapter.EpisodeID,
				URL:             fmt.Sprintf("%sepisodes/%s/", website, chapter.EpisodeID),
				Number:          float32(chapter.Numbering),
				Date:            parseTSMilli(chapter.FirstPublishDate),
				ScanlationGroup: "MangaPlusCreators",
				Volume_:         &volume,
			}
			*chapters = append(*chapters, &c)
		}
		if chaptersDTO.Pagination == nil || !chaptersDTO.Pagination.HasNextPage() {
			break
		}
		page++
	}
	return nil
}

// parseTSMilli: The date of the timestamp, none for 0 (not given).
func parseTSMilli(timestamp int64) metadata.Date {
	if timestamp == 0 {
		return metadata.Date{}
	}
	ts := time.UnixMilli(timestamp)
	return mango.DateOf(ts)
}
