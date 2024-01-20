package mangoprovider

import (
	"encoding/json"

	"github.com/luevano/libmangal"
)

var _ libmangal.Manga = (*Manga)(nil)

type Manga struct {
	Title         string `json:"title"`
	AnilistSearch string `json:"anilist_search"`
	URL           string `json:"url"`
	ID            string `json:"id"`
	Cover         string `json:"cover"`
	Banner        string `json:"banner"`
}

func (m Manga) String() string {
	return m.Title
}

func (m Manga) Info() libmangal.MangaInfo {
	return libmangal.MangaInfo{
		Title: m.Title,
		AnilistSearch: m.AnilistSearch,
		URL: m.URL,
		ID: m.ID,
		Cover: m.Cover,
		Banner: m.Banner,
	}
}

func (m Manga) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Info())
}
