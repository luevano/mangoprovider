package mango

import (
	"encoding/json"

	"github.com/luevano/libmangal"
)

var _ libmangal.Chapter = (*Chapter)(nil)

type Chapter struct {
	Title  string  `json:"title"`
	ID     string  `json:"id"`
	URL    string  `json:"url"`
	Number float32 `json:"number"`

	Volume_ *Volume
}

func (c Chapter) String() string {
	return c.Title
}

func (c Chapter) Info() libmangal.ChapterInfo {
	return libmangal.ChapterInfo{
		Title:  c.Title,
		URL:    c.URL,
		Number: c.Number,
	}
}

func (c Chapter) Volume() libmangal.Volume {
	return c.Volume_
}

func (c Chapter) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Info())
}
