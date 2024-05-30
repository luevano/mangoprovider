package mangoprovider

import (
	"encoding/json"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	"github.com/luevano/mangodex"
)

var _ mangadata.Chapter = (*Chapter)(nil)

type Chapter struct {
	Title           string        `json:"title"`
	ID              string        `json:"id"`
	URL             string        `json:"url"`
	Number          float32       `json:"number"`
	Date            metadata.Date `json:"date"`
	ScanlationGroup string        `json:"scanlation_group"`

	// AtHome is only required for mangadex
	AtHome  *mangodex.AtHomeServer `json:"-"`
	Volume_ *Volume                `json:"-"`
}

func (c *Chapter) String() string {
	return c.Title
}

func (c *Chapter) Info() mangadata.ChapterInfo {
	return mangadata.ChapterInfo{
		Title:           c.Title,
		URL:             c.URL,
		Number:          c.Number,
		Date:            c.Date,
		ScanlationGroup: c.ScanlationGroup,
	}
}

func (c *Chapter) Volume() mangadata.Volume {
	return c.Volume_
}

func (c *Chapter) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Info())
}
