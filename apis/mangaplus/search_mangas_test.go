package mangaplus

import (
	"testing"

	"github.com/luevano/mangoplus"
)

func TestPlusToMangoMangaNoTitle(t *testing.T) {
	p := &plus{}
	if _, err := p.plusToMangoManga(&mangoplus.TitleDetailView{}); err == nil {
		t.Error("wanted an error for details without a title")
	}
	m, err := p.plusToMangoManga(&mangoplus.TitleDetailView{Title: &mangoplus.Title{TitleId: 1, Name: "A"}})
	if err != nil || m.ID != "1" || m.Title != "A" {
		t.Errorf("got %+v, %v", m, err)
	}
}
