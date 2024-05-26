package mangapluscreators

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/philippgille/gokv"
)

func (c *mpc) ChapterPages(ctx context.Context, store gokv.Store, chapter mango.Chapter) ([]libmangal.Page, error) {
	var pages []libmangal.Page

	chapterPages, err := c.client.Page.Get(chapter.ID)
	if err != nil {
		return nil, err
	}

	for _, page := range chapterPages {
		u, err := url.Parse(page.PublicBGImage)
		if err != nil {
			return nil, err
		}
		ext := filepath.Ext(u.Path)
		if !mango.ImageExtensionRegex.MatchString(ext) {
			return nil, fmt.Errorf("invalid page extension: %s (from path %s)", ext, u.Path)
		}

		pageHeaders := map[string]string{
			"Origin":     website,
			"Referer":    chapter.URL,
			"User-Agent": c.userAgent,
		}

		p := mango.Page{
			Extension: ext,
			URL:       u.String(),
			Headers:   pageHeaders,
			Chapter_:  &chapter,
		}

		pages = append(pages, &p)
	}
	return pages, nil
}
