package mangadex

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strconv"

	"github.com/luevano/libmangal/mangadata"
	mango "github.com/luevano/mangoprovider"
)

func (d *dex) MangaVolumes(ctx context.Context, store mango.Store, manga mango.Manga) ([]mangadata.Volume, error) {
	var volumes []mangadata.Volume

	params := url.Values{}
	// none is any language
	if d.filter.Language != "" {
		params.Add("translatedLanguage[]", d.filter.Language)
	}
	// volumes with only retired chapters are left out otherwise
	if d.filter.ShowRetiredChapters {
		params.Set("includeUnavailable", "1")
	}

	cacheID := fmt.Sprintf("%s?%s-%s", manga.ID, params.Encode(), d.filter.String())

	found, err := store.GetVolumes(cacheID, manga, &volumes)
	if err != nil {
		return nil, err
	}
	if found {
		return volumes, nil
	}

	volumeList, err := d.client.Volume.ListContext(ctx, manga.ID, params)
	if err != nil {
		return nil, err
	}

	// the label is sent back as is to get the chapters, e.g. "01" isn't the same as "1"
	var unnumbered []string // "none" and labels that aren't numbers
	latestNumber := float32(0.0)
	for label := range volumeList {
		number64, err := strconv.ParseFloat(label, 32)
		if err != nil {
			unnumbered = append(unnumbered, label)
			continue
		}
		number := float32(number64)
		if number > latestNumber {
			latestNumber = number
		}

		volumes = append(volumes, &mango.Volume{
			Number: number,
			Label:  label,
			Manga_: &manga,
		})
	}

	// listed after the numbered ones
	slices.Sort(unnumbered)
	for i, label := range unnumbered {
		volumes = append(volumes, &mango.Volume{
			Number: float32(int(latestNumber) + 1 + i),
			Label:  label,
			None:   label == "none",
			Manga_: &manga,
		})
	}

	sort.SliceStable(volumes, func(i, j int) bool {
		return volumes[i].Info().Number < volumes[j].Info().Number
	})

	err = store.SetVolumes(cacheID, volumes)
	if err != nil {
		return nil, err
	}

	return volumes, nil
}
