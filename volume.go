package mangoprovider

import (
	"encoding/json"
	"strconv"

	"github.com/luevano/libmangal"
)

var _ libmangal.Volume = (*mangoVolume)(nil)

type mangoVolume struct {
	libmangal.VolumeInfo

	manga *mangoManga
}

func (v mangoVolume) String() string {
	return strconv.Itoa(v.Number)
}

func (v mangoVolume) Info() libmangal.VolumeInfo {
	return v.VolumeInfo
}

func (v mangoVolume) Manga() libmangal.Manga {
	return v.manga
}

func (v mangoVolume) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Info())
}
