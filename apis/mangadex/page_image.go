package mangadex

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/luevano/mangodex"
	mango "github.com/luevano/mangoprovider"
)

func (d *dex) GetPageImage(ctx context.Context, client *http.Client, page mango.Page) ([]byte, error) {
	chapter := page.Chapter_
	d.atHomeMu.Lock()
	atHome := chapter.AtHome
	d.atHomeMu.Unlock()
	if atHome == nil {
		return nil, fmt.Errorf("no MangaDex@Home server, the chapter pages weren't fetched first")
	}

	data := "data"
	if d.options.DataSaver {
		data = "data-saver"
	}
	image, err := atHome.GetChapterPageContext(ctx, data, page.URL, true)
	// a new server wouldn't help with a rate limit, the caller waits for it
	if err == nil || ctx.Err() != nil || errors.Is(err, mangodex.ErrRateLimited) {
		return image, err
	}

	// the MangaDex@Home node failed or its url expired (15 min), MangaDex asks to get a new one
	fresh, freshErr := d.refreshAtHome(ctx, chapter, atHome)
	if freshErr != nil {
		return nil, errors.Join(err, fmt.Errorf("get a new MangaDex@Home server: %w", freshErr))
	}
	if fresh == nil {
		return nil, err
	}
	return fresh.GetChapterPageContext(ctx, data, page.URL, true)
}

// refreshAtHome gets a new MangaDex@Home server for the chapter after failed, once per chapter
// (the server endpoint allows 40 requests a minute): the pages that failed on the same server
// get the same new one, nil when failed is already the new one.
func (d *dex) refreshAtHome(ctx context.Context, chapter *mango.Chapter, failed *mangodex.AtHomeServer) (*mangodex.AtHomeServer, error) {
	d.atHomeMu.Lock()
	defer d.atHomeMu.Unlock()

	if chapter.AtHome != failed {
		return chapter.AtHome, nil
	}
	if d.atHomeRefreshed[chapter.ID] == failed {
		return nil, nil
	}
	fresh, err := d.client.AtHome.GetContext(ctx, chapter.ID, url.Values{})
	if err != nil {
		if ctx.Err() == nil {
			// failed on the first try, don't hammer the endpoint with the other pages
			d.atHomeRefreshed[chapter.ID] = failed
		}
		return nil, err
	}
	chapter.AtHome = fresh
	d.atHomeRefreshed[chapter.ID] = fresh
	return fresh, nil
}
