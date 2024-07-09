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

	atHome, err := d.client.AtHome.Get(chapter.ID, url.Values{})
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
		return nil, fmt.Errorf("no pages for chapter %q (%s); volume %s; manga %q", chapter.String(), chapter.ID, chapter.Volume_.String(), chapter.Volume_.Manga_.String())
	}

	for _, filename := range filenames {
		filenameSplit := strings.Split(filename, ".")
		if len(filenameSplit) != 2 {
			return nil, fmt.Errorf("unexpected error when extracting page extension; chapter %q (%s)", chapter.String(), chapter.ID)
		}

		extension := fmt.Sprintf(".%s", filenameSplit[1])
		if !mango.ImageExtensionRegex.MatchString(extension) {
			return nil, fmt.Errorf("invalid page extension: %s", extension)
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
