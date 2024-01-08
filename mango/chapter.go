package mango

import (
	"encoding/json"

	"github.com/luevano/libmangal"
)

var _ libmangal.Chapter = (*MangoChapter)(nil)

type MangoChapter struct {
	Title string
	URL string
	Number float32

	Volume_ *MangoVolume
}

func (c MangoChapter) String() string {
	return c.Title
}

func (c MangoChapter) Info() libmangal.ChapterInfo {
	return libmangal.ChapterInfo{
		Title: c.Title,
		URL: c.URL,
		Number: c.Number,
	}
}

func (c MangoChapter) Volume() libmangal.Volume {
	return c.Volume_
}

func (c MangoChapter) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Info())
}
