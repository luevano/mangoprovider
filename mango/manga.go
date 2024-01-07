package mango

import (
	"encoding/json"

	"github.com/luevano/libmangal"
)

var _ libmangal.Manga = (*MangoManga)(nil)

type MangoManga struct {
	Title         string `json:"title"`
	AnilistSearch string `json:"anilist_search"`
	URL           string `json:"url"`
	ID            string `json:"id"`
	Cover         string `json:"cover"`
	Banner        string `json:"banner"`
}

func (m MangoManga) String() string {
	return m.Title
}

func (m MangoManga) Info() libmangal.MangaInfo {
	return libmangal.MangaInfo{
		Title: m.Title,
		AnilistSearch: m.AnilistSearch,
		URL: m.URL,
		ID: m.ID,
		Cover: m.Cover,
		Banner: m.Banner,
	}
}

func (m MangoManga) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Info())
}
