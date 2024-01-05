package mango

import (
	"encoding/json"

	"github.com/luevano/libmangal"
)

var _ libmangal.Chapter = (*MangoChapter)(nil)

type MangoChapter struct {
	libmangal.ChapterInfo

	volume *MangoVolume
}

func (c MangoChapter) String() string {
	return c.Title
}

func (c MangoChapter) Info() libmangal.ChapterInfo {
	return c.ChapterInfo
}

func (c MangoChapter) Volume() libmangal.Volume {
	return c.volume
}

func (c MangoChapter) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Info())
}
