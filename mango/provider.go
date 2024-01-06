package mango

import (
	"context"
	"fmt"

	"github.com/luevano/libmangal"
	"github.com/philippgille/gokv"
)

// TODO: need to make another layer for the providerfunctions
// to use MangoX instead of the interfaces from libmangal

var _ libmangal.Provider = (*MangoProvider)(nil)

type ProviderFuncs struct {
	SearchMangas   func(context.Context, gokv.Store, string) ([]libmangal.Manga, error)
	MangaVolumes   func(context.Context, gokv.Store, libmangal.Manga) ([]libmangal.Volume, error)
	VolumeChapters func(context.Context, gokv.Store, libmangal.Volume) ([]libmangal.Chapter, error)
	ChapterPages   func(context.Context, gokv.Store, libmangal.Chapter) ([]libmangal.Page, error)
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
	p.logger.Log(fmt.Sprintf("Fetching volumes for %q", manga))

	return p.Funcs.MangaVolumes(ctx, p.store, manga)
}

func (p *MangoProvider) VolumeChapters(
	ctx context.Context,
	volume libmangal.Volume,
) ([]libmangal.Chapter, error) {
	p.logger.Log(fmt.Sprintf("Fetching chapters for %q", volume))

	return p.Funcs.VolumeChapters(ctx, p.store, volume)
}

func (p *MangoProvider) ChapterPages(
	ctx context.Context,
	chapter libmangal.Chapter,
) ([]libmangal.Page, error) {
	p.logger.Log(fmt.Sprintf("Fetching pages for %q", chapter))

	return p.Funcs.ChapterPages(ctx, p.store, chapter)
}

func (p *MangoProvider) GetPageImage(
	ctx context.Context,
	page libmangal.Page,
) ([]byte, error) {
	p.logger.Log(fmt.Sprintf("Making HTTP GET request for %q", page))

	return nil, fmt.Errorf("unimplemented")
}
