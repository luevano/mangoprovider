package mangoprovider

import (
	"encoding/json"

	"github.com/luevano/libmangal/mangadata"
)

var _ mangadata.Volume = (*Volume)(nil)

type Volume struct {
	Number float32 `json:"number"`
	// Label: The volume as the provider names it, if it's not just the number (e.g. "01", "none").
	Label string `json:"-"`

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
