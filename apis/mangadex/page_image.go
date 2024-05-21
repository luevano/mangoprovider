package mangadex

import (
	"context"
	"fmt"
	"net/http"

	mango "github.com/luevano/mangoprovider"
)

func (d *dex) GetPageImage(ctx context.Context, client *http.Client, page mango.Page) ([]byte, error) {
	atHome := page.Chapter_.AtHome
	if atHome == nil {
		return nil, fmt.Errorf("Chapter's AtHome is nil")
	}

	data := "data"
	if d.filter.MangaDexDataSaver {
		data = "data-saver"
	}
	// TODO: make reporting configurable
	return atHome.GetChapterPage(data, page.URL, true)
}
