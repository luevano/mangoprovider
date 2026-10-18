package mangabox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
)

type chapter struct {
	Name      string   `json:"chapter_name"`
	Slug      string   `json:"chapter_slug"`
	Number    *float32 `json:"chapter_num"`
	UpdatedAt string   `json:"updated_at"`
}

func (m *mangabox) VolumeChapters(ctx context.Context, store mango.Store, volume mango.Volume) ([]mangadata.Chapter, error) {
	var chapters []mangadata.Chapter
	id := volume.Manga_.ID
	if found, err := store.GetChapters(id, volume, &chapters); err != nil || found {
		return chapters, err
	}
	chapters, err := m.chapters(ctx, store, volume)
	if err != nil {
		return nil, err
	}
	return chapters, store.SetChapters(id, chapters)
}

// chapters from the site's api, the manga page only lists the last 50.
func (m *mangabox) chapters(ctx context.Context, store mango.Store, volume mango.Volume) ([]mangadata.Chapter, error) {
	slug := volume.Manga_.ID
	path := "api/manga/" + url.PathEscape(slug) + "/chapters?limit=-1"
	body, site, err := m.get(ctx, store, path)
	if err != nil {
		return nil, err
	}
	// through flaresolverr the json comes back as the browser shows it, in a <pre>
	if bytes.HasPrefix(bytes.TrimSpace(body), []byte("<")) {
		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		body = []byte(doc.Find("pre").First().Text())
	}
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Chapters []chapter `json:"chapters"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("GET /%s: %w", path, err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("GET /%s: %w", path, mango.ErrNotFound)
	}
	taken := map[float32]bool{}
	for _, c := range resp.Data.Chapters {
		if c.Number != nil {
			taken[*c.Number] = true
		}
	}
	var chapters []mangadata.Chapter
	var last float32
	// newest first
	for _, c := range slices.Backward(resp.Data.Chapters) {
		if c.Slug == "" {
			continue
		}
		// without a number it's an extra, right after the previous chapter
		var number float32
		if c.Number != nil {
			number = *c.Number
		} else {
			number = mango.NextFreeExtraNumber(last, taken)
			taken[number] = true
		}
		last = number
		var date metadata.Date
		if t, err := time.Parse(time.RFC3339, c.UpdatedAt); err == nil {
			date = mango.DateOf(t)
		}
		chapters = append(chapters, &mango.Chapter{
			Title:           c.Name,
			ID:              c.Slug,
			URL:             site + "manga/" + slug + "/" + c.Slug,
			Number:          number,
			Date:            date,
			ScanlationGroup: m.info.Name,
			Volume_:         &volume,
		})
	}
	return chapters, nil
}
