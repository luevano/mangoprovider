package mangadex

import (
	"context"
	"fmt"
	"strings"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/mango"
	"github.com/philippgille/gokv"
)

func (d *dex) ChapterPages(ctx context.Context, store gokv.Store, chapter mango.Chapter) ([]libmangal.Page, error) {
	// Note that this doesn't use the store "cache" as mangadex provides a dynamic
	// baseURL/hash/page each time it is consulted
	var pages []libmangal.Page

	data := "data"
	if d.filter.MangaDexDataSaver {
		data = "data-saver"
	}

	mdhome, err := d.client.AtHome.NewMDHomeClient(chapter.ID, data, false)
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

		// TODO: generate random user-agent or use a different static agent
		pageHeaders := map[string]string{
			"Referer":    chapter.URL,
			"Accept":     "image/webp,image/apng,image/*,*/*;q=0.8",
			"User-Agent": mango.UserAgent,
		}

		p := mango.Page{
			Extension: pageExtension,
			URL:       fmt.Sprintf("%s/data/%s/%s", mdhome.BaseURL, mdhome.Hash, page),
			Headers:   pageHeaders,
			Chapter_:  &chapter,
		}

		pages = append(pages, p)
	}
	return pages, nil
}
