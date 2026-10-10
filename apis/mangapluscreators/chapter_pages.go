package mangapluscreators

import (
	"context"

	"github.com/luevano/libmangal/mangadata"
	mango "github.com/luevano/mangoprovider"
)

func (c *mpc) ChapterPages(ctx context.Context, store mango.Store, chapter mango.Chapter) ([]mangadata.Page, error) {
	var pages []mangadata.Page

	chapterPages, err := c.client.Page.GetContext(ctx, chapter.ID)
	if err != nil {
		return nil, err
	}

	for _, page := range chapterPages {
		ext, err := mango.ImageExtension(page.PublicBGImage)
		if err != nil {
			return nil, err
		}

		pageHeaders := map[string]string{
			"Origin":     website,
			"Referer":    chapter.URL,
			"User-Agent": c.options.UserAgent,
		}

		p := mango.Page{
			Ext:      ext,
			URL:      page.PublicBGImage,
			Headers:  pageHeaders,
			Chapter_: &chapter,
		}

		pages = append(pages, &p)
	}
	return pages, nil
}
