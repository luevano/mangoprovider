package asurascans

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
)

type chapter struct {
	Number      float64 `json:"number"`
	Slug        string  `json:"slug"`
	Published   string  `json:"published_at"`
	Premium     bool    `json:"is_premium"`
	EarlyAccess string  `json:"early_access_until"`
}

func (a *asura) VolumeChapters(ctx context.Context, store mango.Store, volume mango.Volume) ([]mangadata.Chapter, error) {
	var chapters []mangadata.Chapter
	id := volume.Manga_.ID
	if found, err := store.GetChapters(id, volume, &chapters); err != nil || found {
		return chapters, err
	}
	chapters, err := a.chapters(ctx, volume)
	if err != nil {
		return nil, err
	}
	return chapters, store.SetChapters(id, chapters)
}

func (a *asura) chapters(ctx context.Context, volume mango.Volume) ([]mangadata.Chapter, error) {
	var chapters []mangadata.Chapter
	manga := volume.Manga_
	var resp struct {
		Data []chapter `json:"data"`
	}
	if err := a.get(ctx, "series/"+url.PathEscape(manga.ID)+"/chapters", &resp); err != nil {
		return nil, err
	}
	// newest first
	for i := len(resp.Data) - 1; i >= 0; i-- {
		c := resp.Data[i]
		// premium chapters are paid early access, free after a few hours or days
		unavailable := ""
		if until, err := time.Parse(time.RFC3339, c.EarlyAccess); c.Premium && err == nil && until.After(time.Now()) {
			unavailable = "early access until " + until.Local().Format(time.DateTime)
		}
		if unavailable != "" && !a.filter.ShowUnavailableChapters {
			continue
		}
		var date metadata.Date
		if t, err := time.Parse(time.RFC3339, c.Published); err == nil {
			date = mango.DateOf(t)
		}
		number := strconv.FormatFloat(c.Number, 'f', -1, 64)
		chapters = append(chapters, &mango.Chapter{
			ID:              c.Slug,
			URL:             manga.URL + "/chapter/" + number,
			Number:          float32(c.Number),
			Date:            date,
			ScanlationGroup: "Asura Scans",
			Unavailable:     unavailable,
			Volume_:         &volume,
		})
	}
	return chapters, nil
}
