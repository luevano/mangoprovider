package mangaplus

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/philippgille/gokv"
)

// TODO: handle case with 0 pages

func (p *plus) ChapterPages(ctx context.Context, store gokv.Store, chapter mango.Chapter) ([]libmangal.Page, error) {
	var pages []libmangal.Page

	chapterPages, err := p.client.Page.Get(chapter.ID, false, "super_high")
	if err != nil {
		return nil, err
	}

	// TODO: handle page extension, currently assuming its .jpg
	for _, page := range chapterPages {

		// if !mango.ImageExtensionRegex.MatchString(pageExtension) {
		// 	return nil, fmt.Errorf("invalid page extension: %s", pageExtension)
		// }

		randUUID, err := uuid.NewRandom()
		if err != nil {
			return nil, err
		}
		pageHeaders := map[string]string{
			"Origin":        website,
			"Referer":       chapter.URL,
			// "Accept":        "image/webp,image/apng,image/*,*/*;q=0.8",
			"User-Agent":    mango.UserAgent,
			"SESSION-TOKEN": randUUID.String(),
		}

		enc := ""
		if page.EncryptionKey != nil {
			enc = fmt.Sprintf("#%s", *page.EncryptionKey)
		}

		p := mango.Page{
			Extension: ".jpg",
			URL:       fmt.Sprintf("%s%s", page.ImageURL, enc),
			Headers:   pageHeaders,
			Chapter_:  &chapter,
		}

		pages = append(pages, &p)
	}
	return pages, nil
}
