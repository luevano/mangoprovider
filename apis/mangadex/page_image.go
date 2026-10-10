package mangadex

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	mango "github.com/luevano/mangoprovider"
)

func (d *dex) GetPageImage(ctx context.Context, client *http.Client, page mango.Page) ([]byte, error) {
	atHome := page.Chapter_.AtHome
	if atHome == nil {
		return nil, fmt.Errorf("no MangaDex@Home server, the chapter pages weren't fetched first")
	}

	data := "data"
	if d.options.DataSaver {
		data = "data-saver"
	}
	// TODO: make reporting configurable
	image, err := atHome.GetChapterPageContext(ctx, data, page.URL, true)
	if err == nil || ctx.Err() != nil {
		return image, err
	}

	// the MangaDex@Home node failed or its url expired (15 min), MangaDex asks to get a new one
	fresh, freshErr := d.client.AtHome.GetContext(ctx, page.Chapter_.ID, url.Values{})
	if freshErr != nil {
		return nil, err
	}
	return fresh.GetChapterPageContext(ctx, data, page.URL, true)
}
