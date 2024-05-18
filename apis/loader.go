package apis

import (
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/apis/mangadex"
	"github.com/luevano/mangoprovider/apis/mangaplus"
)

func Loaders(options mango.Options) []libmangal.ProviderLoader {
	return []libmangal.ProviderLoader{
		mangadex.Loader(options),
		mangaplus.Loader(options),
	}
}
