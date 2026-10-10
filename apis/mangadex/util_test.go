package mangadex

import (
	"testing"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/mangodex"
	mango "github.com/luevano/mangoprovider"
)

func TestGetChapterNum(t *testing.T) {
	five, half := "5", "12.5"
	tests := []struct {
		chapter *string
		number  float32
		title   string
	}{
		{&five, 5, "Chapter 5"},
		{&half, 12.5, "Chapter 12.5"},
		{nil, 0.01, "Chapter 0.01"}, // no number (oneshots, extras)
	}
	for _, tt := range tests {
		chapter := &mangodex.Chapter{Attributes: mangodex.ChapterAttributes{Chapter: tt.chapter}}
		number, title, err := getChapterNum(chapter)
		if err != nil || number != tt.number || title != tt.title {
			t.Errorf("wanted %v, %q; got %v, %q, %v", tt.number, tt.title, number, title, err)
		}
	}
}

func TestGetUniqueChapters(t *testing.T) {
	chapter := func(group, unavailable string) mangadata.Chapter {
		return &mango.Chapter{ScanlationGroup: group, Unavailable: unavailable}
	}
	active, other := chapter("active", ""), chapter("other", "")
	external, available := chapter("active", "external"), chapter("other", "")
	agg := &aggregate{
		chaptersMap: map[string][]mangadata.Chapter{
			"1": {other, active},       // the group with the most uploads wins
			"2": {external, available}, // a downloadable one wins over an unavailable one
		},
		groupsCount: map[string]int{"active": 5, "other": 1},
	}

	chapters, err := getUniqueChapters(agg)
	if err != nil {
		t.Fatal(err)
	}
	if len(chapters) != 2 {
		t.Fatalf("wanted 2 chapters, got %d", len(chapters))
	}
	for _, c := range chapters {
		if c != active && c != available {
			t.Errorf("unexpected chapter picked: %+v", c.Info())
		}
	}
}
