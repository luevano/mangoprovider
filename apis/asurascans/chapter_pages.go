package asurascans

import (
	"context"
	"fmt"
	"net/url"

	"github.com/luevano/libmangal/mangadata"
	mango "github.com/luevano/mangoprovider"
)

type chapterData struct {
	Chapter struct {
		Pages []struct {
			URL string `json:"url"`
		} `json:"pages"`
	} `json:"chapter"`
	Locked     bool   `json:"is_locked"`
	UnlockTime string `json:"unlock_time"`
}

func (a *asura) ChapterPages(ctx context.Context, _ mango.Store, chapter mango.Chapter) ([]mangadata.Page, error) {
	id := chapter.Volume_.Manga_.ID
	var resp struct {
		Data chapterData `json:"data"`
	}
	if err := a.get(ctx, fmt.Sprintf("series/%s/chapters/%s", url.PathEscape(id), url.PathEscape(chapter.ID)), &resp); err != nil {
		return nil, err
	}
	if resp.Data.Locked {
		return nil, fmt.Errorf("chapter %q is early access until %s", chapter.ID, resp.Data.UnlockTime)
	}
	var pages []mangadata.Page
	for _, p := range resp.Data.Chapter.Pages {
		ext, err := mango.ImageExtension(p.URL)
		if err != nil {
			return nil, err
		}
		pages = append(pages, &mango.Page{
			Ext:      ext,
			URL:      p.URL,
			Headers:  map[string]string{"Referer": website, "User-Agent": a.userAgent},
			Chapter_: &chapter,
		})
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("chapter %q has no pages", chapter.ID)
	}
	return pages, nil
}
