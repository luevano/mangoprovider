package mangaplus

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/mangoplus"
	mango "github.com/luevano/mangoprovider"
)

func (p *plus) ChapterPages(ctx context.Context, store mango.Store, chapter mango.Chapter) ([]mangadata.Page, error) {
	var pages []mangadata.Page

	// Will default to "super_high"
	imgQuality := mangoplus.StringToImageQuality(p.options.Quality)
	chapterPages, err := p.client.Page.Get(chapter.ID, false, imgQuality)
	if err != nil {
		return nil, err
	}

	for _, page := range chapterPages {
		u, err := url.Parse(page.ImageUrl)
		if err != nil {
			return nil, err
		}
		ext := filepath.Ext(u.Path)
		if !mango.ImageExtensionRegex.MatchString(ext) {
			return nil, fmt.Errorf("invalid page extension: %s (from path %s)", ext, u.Path)
		}

		// Includes the cookie needed for subscription-only images
		pageHeaders := p.client.ImageHeaders(page.ImageUrl)
		pageHeaders["Origin"] = website
		pageHeaders["Referer"] = chapter.URL

		enc := ""
		if page.EncryptionKey != "" {
			enc = fmt.Sprintf("#%s", page.EncryptionKey)
		}

		p := mango.Page{
			Ext:      ext,
			URL:      fmt.Sprintf("%s%s", page.ImageUrl, enc),
			Headers:  pageHeaders,
			Chapter_: &chapter,
		}

		pages = append(pages, &p)
	}
	return pages, nil
}
