package mangadex

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/luevano/libmangal/mangadata"
	mango "github.com/luevano/mangoprovider"
)

func (d *dex) ChapterPages(ctx context.Context, store mango.Store, chapter mango.Chapter) ([]mangadata.Page, error) {
	var pages []mangadata.Page

	atHome, err := d.client.AtHome.GetContext(ctx, chapter.ID, url.Values{})
	if err != nil {
		return nil, err
	}
	// Required for d.GetPageImage
	chapter.AtHome = atHome

	filenames := atHome.Chapter.Data
	if d.options.DataSaver {
		filenames = atHome.Chapter.DataSaver
	}
	if len(filenames) == 0 {
		return nil, fmt.Errorf("no pages in the MangaDex@Home response: %w", mango.ErrNotFound)
	}

	for _, filename := range filenames {
		filenameSplit := strings.Split(filename, ".")
		if len(filenameSplit) != 2 {
			return nil, fmt.Errorf("invalid image extension in %q", filename)
		}

		extension := fmt.Sprintf(".%s", filenameSplit[1])
		if !mango.ImageExtensionRegex.MatchString(extension) {
			return nil, fmt.Errorf("invalid image extension %q in %q", extension, filename)
		}

		p := mango.Page{
			Ext:      extension,
			URL:      filename,
			Chapter_: &chapter,
		}
		pages = append(pages, &p)
	}
	return pages, nil
}
