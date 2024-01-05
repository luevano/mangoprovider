package mango

import (
	"context"
	"fmt"

	"github.com/luevano/libmangal"
	"github.com/philippgille/gokv"
)

var _ libmangal.Provider = (*MangoProvider)(nil)

type ProviderFuncs struct {
	FnSearchMangas   func(context.Context, string) ([]libmangal.Manga, error)
	FnMangaVolumes   func(context.Context, libmangal.Manga) ([]libmangal.Manga, error)
	FnVolumeChapters func(context.Context, libmangal.Volume) ([]libmangal.Chapter, error)
	FnChapterPages   func(context.Context, libmangal.Chapter) ([]libmangal.Page, error)
	FnGetPageImage   func(context.Context, libmangal.Page) ([]byte, error)
}

type MangoProvider struct {
	libmangal.ProviderInfo
	Options Options

	ProviderFuncs

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

	return p.FnSearchMangas(ctx, query)
}

func (p *MangoProvider) MangaVolumes(
	ctx context.Context,
	manga libmangal.Manga,
) ([]libmangal.Volume, error) {
	p.logger.Log(fmt.Sprintf("Searching manga volumes for manga %q", manga))

	return p.MangaVolumes(ctx, manga)
}

func (p *MangoProvider) VolumeChapters(
	ctx context.Context,
	volume libmangal.Volume,
) ([]libmangal.Chapter, error) {
	p.logger.Log(fmt.Sprintf("Searching manga chapters for volume %q", volume))

	return p.FnVolumeChapters(ctx, volume)
}

func (p *MangoProvider) ChapterPages(
	ctx context.Context,
	chapter libmangal.Chapter,
) ([]libmangal.Page, error) {
	p.logger.Log(fmt.Sprintf("Searching manga pages for chapter %q", chapter))

	return p.FnChapterPages(ctx, chapter)
}

func (p *MangoProvider) GetPageImage(
	ctx context.Context,
	page libmangal.Page,
) ([]byte, error) {
	p.logger.Log(fmt.Sprintf("Searching page image for page %q", page))

	return p.FnGetPageImage(ctx, page)
}
