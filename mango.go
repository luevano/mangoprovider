package mangoprovider

import (
	"encoding/gob"
	"fmt"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/apis/mangadex"
	"github.com/luevano/mangoprovider/mango"
	"github.com/luevano/mangoprovider/scrapers"
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
	}
	loaders = append(loaders, scraper.Loaders(options)...)

	for _, loader := range loaders {
		if loader == nil {
			return nil, fmt.Errorf("failed while loading providers")
		}
	}

	return loaders, nil
}
