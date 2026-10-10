package mangaplus

import "testing"

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
