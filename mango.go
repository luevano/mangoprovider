package mangoprovider

import (
	"encoding/gob"
	"fmt"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/mangadex"
	"github.com/luevano/mangoprovider/mango"
)

// Loaders returns all provider loaders
func Loaders(options mango.Options) ([]libmangal.ProviderLoader, error) {
	// if using gob encoder for the httpstore provided by mangal,
	// then the type/values to be encoded/decoded need to be registered
	gob.Register(mango.MangoManga{})
	gob.Register(mango.MangoVolume{})
	gob.Register(mango.MangoChapter{})
	gob.Register(mango.MangoPage{})

	loaders := []libmangal.ProviderLoader{
		mangadex.Loader(options),
	}

	for _, loader := range loaders {
		if loader == nil {
			return nil, fmt.Errorf("failed while loading providers")
		}
	}

	return loaders, nil
}
