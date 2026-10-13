package mangaplus

import (
	"context"
	"testing"

	mango "github.com/luevano/mangoprovider"
)

func TestParseChapterNumber(t *testing.T) {
	// names as MangaPlus sends them, in order
	names := []string{"#001", "#002", "ex", "#003", "#003", "#004-5"}
	want := []float32{1, 2, 2.5, 3, 3.5, 4.5}

	var last float32
	for i, name := range names {
		got := parseChapterNumber(name, last)
		if got != want[i] {
			t.Errorf("%q after %v: wanted %v, got %v", name, last, want[i], got)
		}
		last = got
	}
}

func TestVolumes(t *testing.T) {
	volumes, err := (&plus{}).MangaVolumes(context.Background(), mango.Store{}, mango.Manga{ID: "1"})
	if err != nil || len(volumes) != 1 || !volumes[0].(*mango.Volume).None {
		t.Errorf("wanted the \"no volumes\" volume, got %v, %v", volumes, err)
	}
}
