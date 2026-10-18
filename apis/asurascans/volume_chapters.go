package asurascans

import (
	"context"
	"net/url"
	"slices"
	"strconv"
	"strings"
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

// earlyAccess: Prefix of the Unavailable of a premium chapter, cached with the RFC 3339 time it
// becomes free and shown with the local one.
const earlyAccess = "early access until "

func (a *asura) VolumeChapters(ctx context.Context, store mango.Store, volume mango.Volume) ([]mangadata.Chapter, error) {
	var chapters []mangadata.Chapter
	id := volume.Manga_.ID
	found, err := store.GetChapters(id, volume, &chapters)
	if err != nil {
		return nil, err
	}
	if !found {
		if chapters, err = a.chapters(ctx, volume); err != nil {
			return nil, err
		}
		if err := store.SetChapters(id, chapters); err != nil {
			return nil, err
		}
	}
	return a.available(chapters, time.Now()), nil
}

// available: The chapters with the early access ones that are free by now made available, the
// unavailable ones left out unless they're shown.
func (a *asura) available(chapters []mangadata.Chapter, now time.Time) []mangadata.Chapter {
	var out []mangadata.Chapter
	for _, chapter := range chapters {
		c := *chapter.(*mango.Chapter)
		if until, err := time.Parse(time.RFC3339, strings.TrimPrefix(c.Unavailable, earlyAccess)); err == nil {
			c.Unavailable = ""
			if until.After(now) {
				c.Unavailable = earlyAccess + until.Local().Format(time.DateTime)
			}
		}
		if c.Unavailable != "" && !a.filter.ShowUnavailableChapters {
			continue
		}
		out = append(out, &c)
	}
	return out
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
	for _, c := range slices.Backward(resp.Data) {
		// premium chapters are paid early access, free after a few hours or days (see available)
		unavailable := ""
		if until, err := time.Parse(time.RFC3339, c.EarlyAccess); c.Premium && err == nil {
			unavailable = earlyAccess + until.UTC().Format(time.RFC3339)
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
