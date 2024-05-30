package mangapluscreators

import (
	"context"
	"fmt"
	"time"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
	"github.com/philippgille/gokv"
)

func (c *mpc) VolumeChapters(ctx context.Context, store gokv.Store, volume mango.Volume) ([]mangadata.Chapter, error) {
	var chapters []mangadata.Chapter

	mangaID := volume.Manga_.ID
	found, err := store.Get(mangaID, &chapters)
	if err != nil {
		return nil, err
	}
	if found {
		mango.Log("found chapters in cache for volume %s", volume.String())
		return chapters, nil
	}

	err = c.searchChapters(&chapters, volume, mangaID)
	if err != nil {
		return nil, err
	}

	err = store.Set(mangaID, chapters)
	if err != nil {
		return nil, err
	}

	return chapters, nil
}

func (c *mpc) searchChapters(chapters *[]mangadata.Chapter, volume mango.Volume, id string) error {
	page := 1
	for {
		chaptersDTO, err := c.client.Chapter.List(volume.Manga_.ID, page)
		if err != nil {
			return err
		}

		pagination := chaptersDTO.Pagination
		if pagination == nil {
			return fmt.Errorf("unexpected error: episodesDto is nil for manga id %q, page %d", id, page)
		}

		// All chapters are assumed to come in order, there is no other way to deal
		// with extra/bonus chapters (if they don't come with a number)
		for _, chapter := range *chaptersDTO.EpisodeList {
			timeStamp := time.UnixMilli(chapter.PublishDate)
			date := metadata.Date{
				Year:  timeStamp.Year(),
				Month: int(timeStamp.Month()),
				Day:   timeStamp.Day(),
			}

			c := mango.Chapter{
				Title:           chapter.EpisodeTitle,
				ID:              chapter.EpisodeID,
				URL:             fmt.Sprintf("%sepisodes/%s/", website, chapter.EpisodeID),
				Number:          float32(chapter.Numbering),
				Date:            date,
				ScanlationGroup: "MangaPlusCreators",
				Volume_:         &volume,
			}
			*chapters = append(*chapters, &c)
		}
		if !pagination.HasNextPage() {
			break
		}
		// TODO: need to double check this
		page += 1
	}
	return nil
}
