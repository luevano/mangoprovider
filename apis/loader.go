package apis

import (
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/apis/mangadex"
)

func Loaders(options mango.Options) []libmangal.ProviderLoader {
	return []libmangal.ProviderLoader{mangadex.Loader(options)}
}
