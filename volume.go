package mangoprovider

import (
	"encoding/json"
	"strconv"

	"github.com/luevano/libmangal"
)

var _ libmangal.Volume = (*Volume)(nil)

type Volume struct {
	Number float32 `json:"number"`

	Manga_ *Manga `json:"-"`
}

func (v *Volume) String() string {
	return strconv.FormatFloat(float64(v.Number), 'f', -1, 32)
}

func (v *Volume) Info() libmangal.VolumeInfo {
	return libmangal.VolumeInfo{
		Number: v.Number,
	}
}

func (v *Volume) Manga() libmangal.Manga {
	return v.Manga_
}

func (v *Volume) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Info())
}
