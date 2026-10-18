package mangaplus

import (
	"context"
	"testing"
	"time"

	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
)

func TestParseChapterNumber(t *testing.T) {
	// names as MangaPlus sends them, in order
	names := []string{"#001", "#002", "ex", "#003", "#003", "#004-5"}
	want := []float32{1, 2, 2.5, 3, 3.5, 4.5}

	taken := map[float32]bool{1: true, 2: true, 3: true, 4.5: true}
	var last float32
	for i, name := range names {
		got := parseChapterNumber(name, last, taken)
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

func TestParseTSSecsIsUTC(t *testing.T) {
	local := time.Local
	time.Local = time.FixedZone("UTC+14", 14*60*60)
	defer func() { time.Local = local }()

	// 2024-01-01 12:00 UTC is already the 2nd at UTC+14
	got := parseTSSecs(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC).Unix())
	if want := (metadata.Date{Year: 2024, Month: 1, Day: 1}); got != want {
		t.Errorf("wanted %v, got %v", want, got)
	}
}
