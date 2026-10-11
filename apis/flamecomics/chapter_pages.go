package flamecomics

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/luevano/libmangal/mangadata"
	mango "github.com/luevano/mangoprovider"
)

type chapterPage struct {
	Token  string `json:"token"`
	Images map[string]struct {
		Name string `json:"name"`
	} `json:"images"`
}

func (f *flame) ChapterPages(ctx context.Context, _ mango.Store, chapter mango.Chapter) ([]mangadata.Page, error) {
	id := chapter.Volume_.Manga_.ID
	var props struct {
		Chapter chapterPage `json:"chapter"`
	}
	if err := f.page(ctx, fmt.Sprintf("series/%s/%s", id, chapter.ID), &props); err != nil {
		return nil, err
	}
	// the images are an object keyed by their position
	keys := make([]int, 0, len(props.Chapter.Images))
	for k := range props.Chapter.Images {
		if n, err := strconv.Atoi(k); err == nil {
			keys = append(keys, n)
		}
	}
	slices.Sort(keys)
	var pages []mangadata.Page
	for _, k := range keys {
		url := fmt.Sprintf("%s%s/%s/%s", cdn, id, chapter.ID, props.Chapter.Images[strconv.Itoa(k)].Name)
		ext, err := mango.ImageExtension(url)
		if err != nil {
			return nil, err
		}
		pages = append(pages, &mango.Page{
			Ext:      ext,
			URL:      url,
			Headers:  map[string]string{"Referer": website, "User-Agent": f.userAgent},
			Chapter_: &chapter,
		})
	}
	return pages, nil
}
