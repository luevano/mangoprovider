package mangaplus

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangoplus"
	mango "github.com/luevano/mangoprovider"
	"github.com/philippgille/gokv"
)

func (p *plus) ChapterPages(ctx context.Context, store gokv.Store, chapter mango.Chapter) ([]libmangal.Page, error) {
	var pages []libmangal.Page

	// Will default to "super_high"
	imgQuality := mangoplus.StringToImageQuality(p.filter.MangaPlusQuality)
	chapterPages, err := p.client.Page.Get(chapter.ID, false, imgQuality)
	if err != nil {
		return nil, err
	}

	for _, page := range chapterPages {
		u, err := url.Parse(page.ImageURL)
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
			"User-Agent": p.userAgent,
		}

		enc := ""
		if page.EncryptionKey != nil {
			enc = fmt.Sprintf("#%s", *page.EncryptionKey)
		}

		p := mango.Page{
			Extension: ext,
			URL:       fmt.Sprintf("%s%s", page.ImageURL, enc),
			Headers:   pageHeaders,
			Chapter_:  &chapter,
		}

		pages = append(pages, &p)
	}
	return pages, nil
}
