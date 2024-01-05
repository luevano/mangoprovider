package mangoprovider

import (
	"encoding/json"

	"github.com/luevano/libmangal"
)

var _ libmangal.Manga = (*mangoManga)(nil)

type mangoManga struct {
	libmangal.MangaInfo
}

func (m mangoManga) String() string {
	return m.Title
}

func (m mangoManga) Info() libmangal.MangaInfo {
	return m.MangaInfo
}

func (m mangoManga) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Info())
}
