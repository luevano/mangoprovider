package mango

import (
	"encoding/json"
	"strconv"

	"github.com/luevano/libmangal"
)

var _ libmangal.Volume = (*MangoVolume)(nil)

type MangoVolume struct {
	Number int `json:"number"`

	Manga_ *MangoManga
}

func (v MangoVolume) String() string {
	return strconv.Itoa(v.Number)
}

func (v MangoVolume) Info() libmangal.VolumeInfo {
	return libmangal.VolumeInfo{
		Number: v.Number,
	}
}

func (v MangoVolume) Manga() libmangal.Manga {
	return v.Manga_
}

func (v MangoVolume) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Info())
}
