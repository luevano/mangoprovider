package flamecomics

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
)

type chapter struct {
	Number  string `json:"chapter"`
	Title   string `json:"title"`
	Token   string `json:"token"`
	Release int64  `json:"release_date"`
}

func (f *flame) VolumeChapters(ctx context.Context, store mango.Store, volume mango.Volume) ([]mangadata.Chapter, error) {
	var chapters []mangadata.Chapter
	id := volume.Manga_.ID
	if found, err := store.GetChapters(id, volume, &chapters); err != nil || found {
		return chapters, err
	}
	var props struct {
		Chapters []chapter `json:"chapters"`
	}
	if err := f.page(ctx, "series/"+id, &props); err != nil {
		return nil, err
	}
	// oldest first, so a chapter without a number goes right after the previous one
	slices.Reverse(props.Chapters)
	var last float32
	for _, c := range props.Chapters {
		number := mango.NextExtraNumber(last)
		if n, err := strconv.ParseFloat(c.Number, 32); err == nil {
			number = float32(n)
		}
		last = number
		var date metadata.Date
		if c.Release != 0 {
			date = mango.DateOf(time.Unix(c.Release, 0).UTC())
		}
		chapters = append(chapters, &mango.Chapter{
			Title:           c.Title,
			ID:              c.Token,
			URL:             fmt.Sprintf("%sseries/%s/%s", website, id, c.Token),
			Number:          number,
			Date:            date,
			ScanlationGroup: "Flame Comics",
			Volume_:         &volume,
		})
	}
	return chapters, store.SetChapters(id, chapters)
}
