package mangoprovider

import (
	"fmt"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/mangadex"
	"github.com/luevano/mangoprovider/mango"
)

// Loaders returns all provider loaders
func Loaders(options mango.Options) ([]libmangal.ProviderLoader, error) {
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
