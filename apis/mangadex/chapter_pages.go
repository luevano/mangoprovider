package mangadex

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/philippgille/gokv"
)

func (d *dex) ChapterPages(ctx context.Context, store gokv.Store, chapter mango.Chapter) ([]libmangal.Page, error) {
	// Note that this doesn't use the store "cache" as mangadex provides a dynamic
	// baseURL/hash/page each time it is consulted
	var pages []libmangal.Page

	atHome, err := d.client.AtHome.Get(chapter.ID, url.Values{})
	if err != nil {
		return nil, err
	}

	data := "data"
	chapterPages := atHome.Chapter.Data
	if d.filter.MangaDexDataSaver {
		chapterPages = atHome.Chapter.DataSaver
		data = "data-saver"
	}

	if len(chapterPages) == 0 {
		return nil, fmt.Errorf("no pages for chapter %q (%s); volume %s; manga %q", chapter.Title, chapter.ID, chapter.Volume_.String(), chapter.Volume_.Manga_.Title)
	}

	for _, page := range chapterPages {
		pageSplit := strings.Split(page, ".")
		if len(pageSplit) != 2 {
			return nil, fmt.Errorf("unexpected error when extracting page extension; chapter %q (%s)", chapter.Title, chapter.ID)
		}
		pageExtension := fmt.Sprintf(".%s", pageSplit[1])

		if !mango.ImageExtensionRegex.MatchString(pageExtension) {
			return nil, fmt.Errorf("invalid page extension: %s", pageExtension)
		}

		pageHeaders := map[string]string{
			"Referer":    chapter.URL,
			"Accept":     "image/webp,image/apng,image/*,*/*;q=0.8",
			"User-Agent": mango.UserAgent,
		}

		p := mango.Page{
			Extension: pageExtension,
			URL:       fmt.Sprintf("%s/%s/%s/%s", atHome.BaseURL, data, atHome.Chapter.Hash, page),
			Headers:   pageHeaders,
			Chapter_:  &chapter,
		}

		pages = append(pages, &p)
	}
	return pages, nil
}
