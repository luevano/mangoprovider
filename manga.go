package mangoprovider

import (
	"encoding/json"
	"fmt"

	"github.com/luevano/libmangal"
)

var (
	_ libmangal.MangaWithSeriesJSON = (*Manga)(nil)
	_ libmangal.Manga               = (*Manga)(nil)
)

type Manga struct {
	Title         string `json:"title"`
	AnilistSearch string `json:"anilist_search"`
	URL           string `json:"url"`
	ID            string `json:"id"`
	Cover         string `json:"cover"`
	Banner        string `json:"banner"`

	AnilistSet_ bool                   `json:"-"`
	Anilist_    libmangal.AnilistManga `json:"-"`
}

func (m *Manga) String() string {
	return m.Title
}

func (m *Manga) Info() libmangal.MangaInfo {
	return libmangal.MangaInfo{
		Title:         m.Title,
		AnilistSearch: m.AnilistSearch,
		URL:           m.URL,
		ID:            m.ID,
		Cover:         m.Cover,
		Banner:        m.Banner,
	}
}

func (m *Manga) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Info())
}

func (m *Manga) AnilistManga() (libmangal.AnilistManga, error) {
	if m.AnilistSet_ {
		return m.Anilist_, nil
	} else {
		return libmangal.AnilistManga{}, fmt.Errorf("AnilistManga is not set")
	}
}

func (m *Manga) SetAnilistManga(anilist libmangal.AnilistManga) {
	m.Anilist_ = anilist
	m.AnilistSet_ = true
}

func (m *Manga) SeriesJSON() (libmangal.SeriesJSON, bool, error) {
	if !m.AnilistSet_ {
		Log(fmt.Sprintf("manga %q doesn't contain anilist data", m.Title))
		return libmangal.SeriesJSON{}, false, nil
	}

	return m.Anilist_.SeriesJSON(), true, nil
}
