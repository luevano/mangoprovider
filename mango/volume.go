package mango

import (
	"encoding/json"
	"strconv"

	"github.com/luevano/libmangal"
)

var _ libmangal.Volume = (*MangoVolume)(nil)

type MangoVolume struct {
	libmangal.VolumeInfo

	manga *MangoManga
}

func (v MangoVolume) String() string {
	return strconv.Itoa(v.Number)
}

func (v MangoVolume) Info() libmangal.VolumeInfo {
	return v.VolumeInfo
}

func (v MangoVolume) Manga() libmangal.Manga {
	return v.manga
}

func (v MangoVolume) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Info())
}
