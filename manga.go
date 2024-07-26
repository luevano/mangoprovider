package mangoprovider

import (
	"encoding/json"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
)

var _ mangadata.Manga = (*Manga)(nil)

type Manga struct {
	Title  string `json:"title"`
	URL    string `json:"url"`
	ID     string `json:"id"`
	Cover  string `json:"cover"`
	Banner string `json:"banner"`

	Metadata_ *metadata.Metadata `json:"-"`
}

func (m *Manga) String() string {
	return m.Title
}

func (m *Manga) Info() mangadata.MangaInfo {
	return mangadata.MangaInfo{
		Title:  m.Title,
		URL:    m.URL,
		ID:     m.ID,
		Cover:  m.Cover,
		Banner: m.Banner,
	}
}

func (m *Manga) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Info())
}

func (m *Manga) Metadata() metadata.Metadata {
	return *m.Metadata_
}

func (m *Manga) SetMetadata(metadata metadata.Metadata) {
	*m.Metadata_ = metadata
}
