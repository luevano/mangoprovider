package mango

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/luevano/libmangal"
	"github.com/philippgille/gokv"
)

var _ libmangal.Provider = (*Provider)(nil)

type ProviderFuncs struct {
	SearchMangas   func(context.Context, gokv.Store, string) ([]libmangal.Manga, error)
	MangaVolumes   func(context.Context, gokv.Store, Manga) ([]libmangal.Volume, error)
	VolumeChapters func(context.Context, gokv.Store, Volume) ([]libmangal.Chapter, error)
	ChapterPages   func(context.Context, gokv.Store, Chapter) ([]libmangal.Page, error)
	GetPageImage   func(context.Context, Page) ([]byte, error)
}

type Provider struct {
	libmangal.ProviderInfo
	Options Options
	Funcs   ProviderFuncs

	// TODO: the logger is not actually being used anywhere
	store  gokv.Store
	logger *libmangal.Logger
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

func (p *Provider) SetLogger(logger *libmangal.Logger) {
	p.logger = logger
}

func (p *Provider) SearchMangas(
	ctx context.Context,
	query string,
) ([]libmangal.Manga, error) {
	p.logger.Log(fmt.Sprintf("Searching mangas with %q", query))

	return p.Funcs.SearchMangas(ctx, p.store, query)
}

func (p *Provider) MangaVolumes(
	ctx context.Context,
	manga libmangal.Manga,
) ([]libmangal.Volume, error) {
	m, ok := manga.(Manga)
	if !ok {
		return nil, fmt.Errorf("unexpected manga type: %T", manga)
	}

	p.logger.Log(fmt.Sprintf("Fetching volumes for %q", m))
	return p.Funcs.MangaVolumes(ctx, p.store, m)
}

func (p *Provider) VolumeChapters(
	ctx context.Context,
	volume libmangal.Volume,
) ([]libmangal.Chapter, error) {
	v, ok := volume.(Volume)
	if !ok {
		return nil, fmt.Errorf("unexpected volume type: %T", volume)
	}

	p.logger.Log(fmt.Sprintf("Fetching chapters for %q", v))
	return p.Funcs.VolumeChapters(ctx, p.store, v)
}

func (p *Provider) ChapterPages(
	ctx context.Context,
	chapter libmangal.Chapter,
) ([]libmangal.Page, error) {
	c, ok := chapter.(Chapter)
	if !ok {
		return nil, fmt.Errorf("unexpected chapter type: %T", chapter)
	}

	p.logger.Log(fmt.Sprintf("Fetching pages for %q", c))
	return p.Funcs.ChapterPages(ctx, p.store, c)
}

func (p *Provider) GetPageImage(
	ctx context.Context,
	page libmangal.Page,
) ([]byte, error) {
	page_, ok := page.(Page)
	if !ok {
		return nil, fmt.Errorf("unexpected page type: %T", page)
	}

	p.logger.Log(fmt.Sprintf("Making HTTP GET request for %q", page_.URL))
	if p.Funcs.GetPageImage != nil {
		return p.Funcs.GetPageImage(ctx, page_)
	} else {
		return p.GenericGetPageImage(ctx, page_)
	}
}

func (p *Provider) GenericGetPageImage(
	ctx context.Context,
	page Page,
) ([]byte, error) {
	p.logger.Log("Making request using generic getter.")
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

	response, err := p.Options.HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	p.logger.Log("Got response")

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	image, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return image, nil
}
