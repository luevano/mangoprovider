package mangoprovider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/logger"
	"github.com/luevano/libmangal/mangadata"
)

var (
	_ libmangal.Provider          = (*Provider)(nil)
	_ libmangal.AccountProvider   = (*Provider)(nil)
	_ libmangal.MangaByIDProvider = (*Provider)(nil)
)

type Functions struct {
	SearchMangas   func(context.Context, Store, string) ([]mangadata.Manga, error)
	MangaVolumes   func(context.Context, Store, Manga) ([]mangadata.Volume, error)
	VolumeChapters func(context.Context, Store, Volume) ([]mangadata.Chapter, error)
	ChapterPages   func(context.Context, Store, Chapter) ([]mangadata.Page, error)
	GetPageImage   func(context.Context, *http.Client, Page) ([]byte, error)
	// MangaByID: Optional, the manga with the given id, found false if there's none.
	MangaByID func(context.Context, Store, string) (mangadata.Manga, bool, error)
	// Close: Optional, frees what the provider holds (e.g. a headless browser).
	Close func() error
	// Account: Optional, the user's account (e.g. a subscription).
	Account func(context.Context) (libmangal.Account, bool, error)
}

type Provider struct {
	libmangal.ProviderInfo
	Options Options
	F       Functions

	client *http.Client
	store  Store
	log    *Logger
}

func (p *Provider) String() string {
	return p.Name
}

func (p *Provider) Close() error {
	var err error
	if p.F.Close != nil {
		err = p.F.Close()
	}
	return errors.Join(err, p.store.Close())
}

// Account: The user's account, found false if the provider has none.
func (p *Provider) Account(ctx context.Context) (libmangal.Account, bool, error) {
	if p.F.Account == nil {
		return libmangal.Account{}, false, nil
	}
	account, found, err := p.F.Account(ctx)
	if err != nil {
		return account, found, fmt.Errorf("%s: account: %w", p.ProviderInfo.ID, err)
	}
	return account, found, nil
}

func (p *Provider) Info() libmangal.ProviderInfo {
	return p.ProviderInfo
}

func (p *Provider) SetLogger(logger *logger.Logger) {
	p.log.SetLogger(logger)
}

func (p *Provider) SearchMangas(
	ctx context.Context,
	query string,
) ([]mangadata.Manga, error) {
	p.log.Log("searching mangas with query %q", query)

	mangas, err := p.F.SearchMangas(ctx, p.store, query)
	if err != nil {
		return nil, fmt.Errorf("%s: search %q: %w", p.ProviderInfo.ID, query, err)
	}
	return mangas, nil
}

// MangaByID: The manga with the given id, found false if there's none. The error matches
// libmangal.ErrMangaByIDUnsupported if the provider can't get mangas by id.
func (p *Provider) MangaByID(ctx context.Context, id string) (mangadata.Manga, bool, error) {
	if p.F.MangaByID == nil {
		return nil, false, fmt.Errorf("%s: %w", p.ProviderInfo.ID, libmangal.ErrMangaByIDUnsupported)
	}
	p.log.Log("getting manga with id %q", id)
	manga, found, err := p.F.MangaByID(ctx, p.store, id)
	if err != nil {
		return nil, false, fmt.Errorf("%s: manga %q: %w", p.ProviderInfo.ID, id, err)
	}
	if !found || manga == nil {
		return nil, false, nil
	}
	return manga, true, nil
}

// SearchByID: The result of a MangaByID as a search result, for "mid:" queries.
func SearchByID(manga mangadata.Manga, found bool, err error) ([]mangadata.Manga, error) {
	if err != nil || !found {
		return nil, err
	}
	return []mangadata.Manga{manga}, nil
}

func (p *Provider) MangaVolumes(
	ctx context.Context,
	manga mangadata.Manga,
) ([]mangadata.Volume, error) {
	m, ok := manga.(*Manga)
	if !ok {
		return nil, fmt.Errorf("expected a mangoprovider manga, got %T", manga)
	}

	p.log.Log("fetching volumes for manga %q", m)
	volumes, err := p.F.MangaVolumes(ctx, p.store, *m)
	if err != nil {
		return nil, fmt.Errorf("%s: volumes of %q: %w", p.ProviderInfo.ID, m, err)
	}
	return volumes, nil
}

func (p *Provider) VolumeChapters(
	ctx context.Context,
	volume mangadata.Volume,
) ([]mangadata.Chapter, error) {
	v, ok := volume.(*Volume)
	if !ok {
		return nil, fmt.Errorf("expected a mangoprovider volume, got %T", volume)
	}

	p.log.Log("fetching chapters for volume %s", v.String())
	chapters, err := p.F.VolumeChapters(ctx, p.store, *v)
	if err != nil {
		return nil, fmt.Errorf("%s: chapters of %q vol %s: %w", p.ProviderInfo.ID, v.Manga_, v, err)
	}
	return chapters, nil
}

func (p *Provider) ChapterPages(
	ctx context.Context,
	chapter mangadata.Chapter,
) ([]mangadata.Page, error) {
	c, ok := chapter.(*Chapter)
	if !ok {
		return nil, fmt.Errorf("expected a mangoprovider chapter, got %T", chapter)
	}

	p.log.Log("fetching pages for chapter %q", c)
	pages, err := p.F.ChapterPages(ctx, p.store, *c)
	if err != nil {
		return nil, fmt.Errorf("%s: pages of chapter %q: %w", p.ProviderInfo.ID, c, err)
	}
	return pages, nil
}

func (p *Provider) GetPageImage(
	ctx context.Context,
	page mangadata.Page,
) ([]byte, error) {
	page_, ok := page.(*Page)
	if !ok {
		return nil, fmt.Errorf("expected a mangoprovider page, got %T", page)
	}

	getPageImage := GenericGetPageImage
	if p.F.GetPageImage != nil {
		getPageImage = p.F.GetPageImage
	}
	image, err := getPageImage(ctx, p.client, *page_)
	if err != nil {
		return nil, fmt.Errorf("%s: image for chapter %q: %w", p.ProviderInfo.ID, page_.Chapter_, err)
	}
	return image, nil
}

func GenericGetPageImage(
	ctx context.Context,
	client *http.Client,
	page Page,
) ([]byte, error) {
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

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %d %s", page.URL, response.StatusCode, strings.ToLower(http.StatusText(response.StatusCode)))
	}

	image, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return image, nil
}
