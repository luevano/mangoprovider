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
	var pages []libmangal.Page

	atHome, err := d.client.AtHome.Get(chapter.ID, url.Values{})
	if err != nil {
		return nil, err
	}
	// Required for d.GetPageImage
	chapter.AtHome = atHome

	filenames := atHome.Chapter.Data
	if d.filter.MangaDexDataSaver {
		filenames = atHome.Chapter.DataSaver
	}
	if len(filenames) == 0 {
		return nil, fmt.Errorf("no pages for chapter %q (%s); volume %s; manga %q", chapter.Title, chapter.ID, chapter.Volume_.String(), chapter.Volume_.Manga_.Title)
	}

	for _, filename := range filenames {
		filenameSplit := strings.Split(filename, ".")
		if len(filenameSplit) != 2 {
			return nil, fmt.Errorf("unexpected error when extracting page extension; chapter %q (%s)", chapter.Title, chapter.ID)
		}

		extension := fmt.Sprintf(".%s", filenameSplit[1])
		if !mango.ImageExtensionRegex.MatchString(extension) {
			return nil, fmt.Errorf("invalid page extension: %s", extension)
		}

		p := mango.Page{
			Extension: extension,
			URL:       filename,
			Chapter_:  &chapter,
		}
		pages = append(pages, &p)
	}
	return pages, nil
}
