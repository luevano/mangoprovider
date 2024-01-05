package mangoprovider

import (
	"encoding/json"

	"github.com/luevano/libmangal"
)

var _ libmangal.Chapter = (*mangoChapter)(nil)

type mangoChapter struct {
	libmangal.ChapterInfo

	volume *mangoVolume
}

func (c mangoChapter) String() string {
	return c.Title
}

func (c mangoChapter) Info() libmangal.ChapterInfo {
	return c.ChapterInfo
}

func (c mangoChapter) Volume() libmangal.Volume {
	return c.volume
}

func (c mangoChapter) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Info())
}
