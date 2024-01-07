package mango

import (
	"context"
	"fmt"

	"github.com/luevano/libmangal"
	"github.com/philippgille/gokv"
)

var _ libmangal.Provider = (*MangoProvider)(nil)

type ProviderFuncs struct {
	SearchMangas   func(context.Context, gokv.Store, string) ([]libmangal.Manga, error)
	MangaVolumes   func(context.Context, gokv.Store, MangoManga) ([]libmangal.Volume, error)
	VolumeChapters func(context.Context, gokv.Store, MangoVolume) ([]libmangal.Chapter, error)
	ChapterPages   func(context.Context, gokv.Store, MangoChapter) ([]libmangal.Page, error)
}

type MangoProvider struct {
	libmangal.ProviderInfo
	Options Options
	Funcs   ProviderFuncs

	store  gokv.Store
	logger *libmangal.Logger
}

func (p *MangoProvider) String() string {
	return p.Name
}

func (p *MangoProvider) Close() error {
	return p.store.Close()
}

func (p *MangoProvider) Info() libmangal.ProviderInfo {
	return p.ProviderInfo
}

func (p *MangoProvider) SetLogger(logger *libmangal.Logger) {
	p.logger = logger
}

func (p *MangoProvider) SearchMangas(
	ctx context.Context,
	query string,
) ([]libmangal.Manga, error) {
	p.logger.Log(fmt.Sprintf("Searching mangas with %q", query))

	return p.Funcs.SearchMangas(ctx, p.store, query)
}

func (p *MangoProvider) MangaVolumes(
	ctx context.Context,
	manga libmangal.Manga,
) ([]libmangal.Volume, error) {
	m, ok := manga.(MangoManga)
	if !ok {
		return nil, fmt.Errorf("unexpected manga type: %T", manga)
	}

	p.logger.Log(fmt.Sprintf("Fetching volumes for %q", m))
	return p.Funcs.MangaVolumes(ctx, p.store, m)
}

func (p *MangoProvider) VolumeChapters(
	ctx context.Context,
	volume libmangal.Volume,
) ([]libmangal.Chapter, error) {
	v, ok := volume.(MangoVolume)
	if !ok {
		return nil, fmt.Errorf("unexpected volume type: %T", volume)
	}

	p.logger.Log(fmt.Sprintf("Fetching chapters for %q", v))
	return p.Funcs.VolumeChapters(ctx, p.store, v)
}

func (p *MangoProvider) ChapterPages(
	ctx context.Context,
	chapter libmangal.Chapter,
) ([]libmangal.Page, error) {
	c, ok := chapter.(MangoChapter)
	if !ok {
		return nil, fmt.Errorf("unexpected chapter type: %T", chapter)
	}

	p.logger.Log(fmt.Sprintf("Fetching pages for %q", c))
	return p.Funcs.ChapterPages(ctx, p.store, c)
}

func (p *MangoProvider) GetPageImage(
	ctx context.Context,
	page libmangal.Page,
) ([]byte, error) {
	page_, ok := page.(MangoPage)
	if !ok {
		return nil, fmt.Errorf("unexpected page type: %T", page)
	}

	p.logger.Log(fmt.Sprintf("Making HTTP GET request for %q", page_.URL))
	return p.getPageImage(ctx, page_)
}

func (p *MangoProvider) getPageImage(
	ctx context.Context,
	page MangoPage,
) ([]byte, error) {
	return nil, fmt.Errorf("unimplemented")
}
