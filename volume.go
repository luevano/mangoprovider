package mangoprovider

import (
	"encoding/json"

	"github.com/luevano/libmangal/mangadata"
)

var _ mangadata.Volume = (*Volume)(nil)

type Volume struct {
	Number float32 `json:"number"`

	None   bool   `json:"-"`
	Manga_ *Manga `json:"-"`
}

func (v *Volume) String() string {
	return FormattedFloat(v.Number)
}

func (v *Volume) Info() mangadata.VolumeInfo {
	return mangadata.VolumeInfo{
		Number: v.Number,
	}
}

func (v *Volume) Manga() mangadata.Manga {
	return v.Manga_
}

func (v *Volume) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Info())
}
