package mangadex

import (
	"context"
	"fmt"
	"strings"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/mango"
	"github.com/philippgille/gokv"
)

func (d *Dex) ChapterPages(ctx context.Context, store gokv.Store, chapter mango.MangoChapter) ([]libmangal.Page, error) {
	// Note that this doesn't use the store "cache" as mangadex provides a dynamic
	// baseURL/hash/page each time it is consulted
	var pages []libmangal.Page

	// TODO: add an option to select the "quality" ("data" or "data-saver")
	mdhome, err := d.client.AtHome.NewMDHomeClient(chapter.ID, "data", false)
	if err != nil {
		return nil, err
	}

	if len(mdhome.Pages) == 0 {
		return nil, fmt.Errorf("no pages for chapter %q (%s); volume %q; manga %q", chapter.Title, chapter.ID, chapter.Volume_.Number, chapter.Volume_.Manga_.Title)
	}

	for _, page := range mdhome.Pages {
		pageSplit := strings.Split(page, ".")
		if len(pageSplit) != 2 {
			return nil, fmt.Errorf("unexpected error when extracting page extension; chapter %q (%s)", chapter.Title, chapter.ID)
		}
		pageExtension := fmt.Sprintf(".%s", pageSplit[1])

		if !mango.ImageExtensionRegex.MatchString(pageExtension) {
			return nil, fmt.Errorf("invalid page extension: %s", pageExtension)
		}

		pageHeaders := map[string]string{
			"Referer": chapter.URL,
			"Accept":  "image/webp,image/apng,image/*,*/*;q=0.8",
			// TODO: generate random user-agent or use a different static agent
			"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36",
		}

		p := mango.MangoPage{
			Extension: pageExtension,
			URL:       fmt.Sprintf("%s/data/%s/%s", mdhome.BaseURL, mdhome.Hash, page),
			Headers:   pageHeaders,
			Chapter_:  &chapter,
		}

		pages = append(pages, p)
	}
	return pages, nil
}
