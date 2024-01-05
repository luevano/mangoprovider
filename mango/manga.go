package mango

import (
	"encoding/json"

	"github.com/luevano/libmangal"
)

var _ libmangal.Manga = (*MangoManga)(nil)

type MangoManga struct {
	libmangal.MangaInfo
}

func (m MangoManga) String() string {
	return m.Title
}

func (m MangoManga) Info() libmangal.MangaInfo {
	return m.MangaInfo
}

func (m MangoManga) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Info())
}
