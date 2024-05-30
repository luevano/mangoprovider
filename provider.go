package mangoprovider

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/logger"
	"github.com/luevano/libmangal/mangadata"
	"github.com/philippgille/gokv"
)

var _ libmangal.Provider = (*Provider)(nil)

type Functions struct {
	SearchMangas   func(context.Context, gokv.Store, string) ([]mangadata.Manga, error)
	MangaVolumes   func(context.Context, gokv.Store, Manga) ([]mangadata.Volume, error)
	VolumeChapters func(context.Context, gokv.Store, Volume) ([]mangadata.Chapter, error)
	ChapterPages   func(context.Context, gokv.Store, Chapter) ([]mangadata.Page, error)
	GetPageImage   func(context.Context, *http.Client, Page) ([]byte, error)
}

type Provider struct {
	libmangal.ProviderInfo
	Options Options
	F       Functions

	client *http.Client
	store  gokv.Store
}

func (p *Provider) String() string {
	return p.Name
}

func (p *Provider) Close() error {
	return p.store.Close()
}

func (p *Provider) Info() libmangal.ProviderInfo {
	return p.ProviderInfo
}

func (p *Provider) SetLogger(newLogger *logger.Logger) {
	logger_ = newLogger
}

func (p *Provider) SearchMangas(
	ctx context.Context,
	query string,
) ([]mangadata.Manga, error) {
	Log("searching mangas with query %q", query)

	return p.F.SearchMangas(ctx, p.store, query)
}

func (p *Provider) MangaVolumes(
	ctx context.Context,
	manga mangadata.Manga,
) ([]mangadata.Volume, error) {
	m, ok := manga.(*Manga)
	if !ok {
		return nil, fmt.Errorf("unexpected manga type: %T", manga)
	}

	Log("fetching volumes for manga %q", m)
	return p.F.MangaVolumes(ctx, p.store, *m)
}

func (p *Provider) VolumeChapters(
	ctx context.Context,
	volume mangadata.Volume,
) ([]mangadata.Chapter, error) {
	v, ok := volume.(*Volume)
	if !ok {
		return nil, fmt.Errorf("unexpected volume type: %T", volume)
	}

	Log("fetching chapters for volume %s", v.String())
	return p.F.VolumeChapters(ctx, p.store, *v)
}

func (p *Provider) ChapterPages(
	ctx context.Context,
	chapter mangadata.Chapter,
) ([]mangadata.Page, error) {
	c, ok := chapter.(*Chapter)
	if !ok {
		return nil, fmt.Errorf("unexpected chapter type: %T", chapter)
	}

	Log("fetching pages for chapter %q", c)
	return p.F.ChapterPages(ctx, p.store, *c)
}

func (p *Provider) GetPageImage(
	ctx context.Context,
	page mangadata.Page,
) ([]byte, error) {
	page_, ok := page.(*Page)
	if !ok {
		return nil, fmt.Errorf("unexpected page type: %T", page)
	}

	// Log(fmt.Sprintf("Making HTTP GET request for %q", page_.URL))
	if p.F.GetPageImage != nil {
		return p.F.GetPageImage(ctx, p.client, *page_)
	} else {
		return GenericGetPageImage(ctx, p.client, *page_)
	}
}

func GenericGetPageImage(
	ctx context.Context,
	client *http.Client,
	page Page,
) ([]byte, error) {
	// Log("Making request using generic getter.")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, page.URL, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range page.Headers {
		request.Header.Set(key, value)
	}

	for key, value := range page.Cookies {
		request.AddCookie(&http.Cookie{Name: key, Value: value})
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	// Log("Got response")

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	image, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return image, nil
}
