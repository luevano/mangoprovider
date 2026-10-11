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
	for _, c := range props.Chapters {
		number, err := strconv.ParseFloat(c.Number, 32)
		if err != nil {
			continue
		}
		date := time.Unix(c.Release, 0)
		chapters = append(chapters, &mango.Chapter{
			Title:           c.Title,
			ID:              c.Token,
			URL:             fmt.Sprintf("%sseries/%s/%s", website, id, c.Token),
			Number:          float32(number),
			Date:            metadata.Date{Year: date.Year(), Month: int(date.Month()), Day: date.Day()},
			ScanlationGroup: "Flame Comics",
			Volume_:         &volume,
		})
	}
	// newest first on the site
	slices.Reverse(chapters)
	return chapters, store.SetChapters(id, chapters)
}
