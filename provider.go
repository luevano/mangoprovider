package mangoprovider

import (
	"context"
	"fmt"

	"github.com/luevano/libmangal"
	"github.com/philippgille/gokv"
)

var _ libmangal.Provider = (*mangoProvider)(nil)

type mangoProvider struct {
	libmangal.ProviderInfo
	options Options
	store   gokv.Store
	logger  *libmangal.Logger
}

func (p *mangoProvider) String() string {
	return p.Name
}

func (p *mangoProvider) Close() error {
	return p.store.Close()
}

func (p *mangoProvider) Info() libmangal.ProviderInfo {
	return p.ProviderInfo
}

func (p *mangoProvider) SetLogger(logger *libmangal.Logger) {
	p.logger = logger
}

func (p *mangoProvider) SearchMangas(
	ctx context.Context,
	query string,
) ([]libmangal.Manga, error) {
	p.logger.Log(fmt.Sprintf("Searching mangas with %q", query))

	return []libmangal.Manga{}, nil
}

func (p *mangoProvider) MangaVolumes(
	ctx context.Context,
	manga libmangal.Manga,
) ([]libmangal.Volume, error) {
	p.logger.Log(fmt.Sprintf("Searching manga volumes for manga %q", manga))

	return []libmangal.Volume{}, nil
}

func (p *mangoProvider) VolumeChapters(
	ctx context.Context,
	volume libmangal.Volume,
) ([]libmangal.Chapter, error) {
	p.logger.Log(fmt.Sprintf("Searching manga chapters for volume %q", volume))

	return []libmangal.Chapter{}, nil
}

func (p *mangoProvider) ChapterPages(
	ctx context.Context,
	chapter libmangal.Chapter,
) ([]libmangal.Page, error) {
	p.logger.Log(fmt.Sprintf("Searching manga pages for chapter %q", chapter))

	return []libmangal.Page{}, nil
}

func (p *mangoProvider) GetPageImage(
	ctx context.Context,
	page libmangal.Page,
) ([]byte, error) {
	p.logger.Log(fmt.Sprintf("Searching page image for page %q", page))

	return []byte{}, nil
}
