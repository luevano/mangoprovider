package apis

import (
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/apis/asurascans"
	"github.com/luevano/mangoprovider/apis/flamecomics"
	"github.com/luevano/mangoprovider/apis/mangabox"
	"github.com/luevano/mangoprovider/apis/mangadex"
	"github.com/luevano/mangoprovider/apis/mangaplus"
	"github.com/luevano/mangoprovider/apis/mangapluscreators"
)

func Loaders(options mango.Options) []libmangal.ProviderLoader {
	loaders := []libmangal.ProviderLoader{
		mangadex.Loader(options),
		mangaplus.Loader(options),
		mangapluscreators.Loader(options),
		flamecomics.Loader(options),
		asurascans.Loader(options),
	}
	return append(loaders, mangabox.Loaders(options)...)
}
