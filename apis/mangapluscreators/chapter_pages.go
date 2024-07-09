package mangapluscreators

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/luevano/libmangal/mangadata"
	mango "github.com/luevano/mangoprovider"
)

func (c *mpc) ChapterPages(ctx context.Context, store mango.Store, chapter mango.Chapter) ([]mangadata.Page, error) {
	var pages []mangadata.Page

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
			"User-Agent": c.options.UserAgent,
		}

		p := mango.Page{
			Ext:      ext,
			URL:      u.String(),
			Headers:  pageHeaders,
			Chapter_: &chapter,
		}

		pages = append(pages, &p)
	}
	return pages, nil
}
