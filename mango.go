package mangoprovider

import (
	"encoding/gob"
	"fmt"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/asurascans"
	"github.com/luevano/mangoprovider/flamescans"
	"github.com/luevano/mangoprovider/mangadex"
	"github.com/luevano/mangoprovider/manganato"
	"github.com/luevano/mangoprovider/manganelo"
	"github.com/luevano/mangoprovider/mangapill"
	"github.com/luevano/mangoprovider/mango"
)

// Loaders returns all provider loaders
func Loaders(options mango.Options) ([]libmangal.ProviderLoader, error) {
	// if using gob encoder for the httpstore provided by mangal,
	// then the type/values to be encoded/decoded need to be registered
	gob.Register(mango.Manga{})
	gob.Register(mango.Volume{})
	gob.Register(mango.Chapter{})
	gob.Register(mango.Page{})

	loaders := []libmangal.ProviderLoader{
		mangadex.Loader(options),
		mangapill.Loader(options),
		asurascans.Loader(options),
		flamescans.Loader(options),
		manganato.Loader(options),
		manganelo.Loader(options),
	}

	for _, loader := range loaders {
		if loader == nil {
			return nil, fmt.Errorf("failed while loading providers")
		}
	}

	return loaders, nil
}
