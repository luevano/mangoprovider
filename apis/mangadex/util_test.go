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
		chapter  *string
		number   float32
		numbered bool
	}{
		{&five, 5, true},
		{&half, 12.5, true},
		{nil, 0, false}, // no number (oneshots, extras)
	}
	for _, tt := range tests {
		chapter := &mangodex.Chapter{Attributes: mangodex.ChapterAttributes{Chapter: tt.chapter}}
		number, numbered, err := getChapterNum(chapter)
		if err != nil || number != tt.number || numbered != tt.numbered {
			t.Errorf("wanted %v, %v; got %v, %v, %v", tt.number, tt.numbered, number, numbered, err)
		}
	}
}

func TestNumberExtraChapters(t *testing.T) {
	one, extra, two := &mango.Chapter{Number: 1}, &mango.Chapter{Title: "Side story"}, &mango.Chapter{Number: 2}
	extra2, extra3 := &mango.Chapter{}, &mango.Chapter{}
	numberExtraChapters([]mangadata.Chapter{one, extra, two, extra2, extra3}, map[*mango.Chapter]bool{extra: true, extra2: true, extra3: true})

	for _, tt := range []struct {
		chapter *mango.Chapter
		number  float32
		title   string
	}{
		{one, 1, "Chapter 1"},
		{extra, 1.5, "Side story"},
		{two, 2, "Chapter 2"},
		{extra2, 2.5, "Chapter 2.5"},
		{extra3, 2.6, "Chapter 2.6"},
	} {
		if tt.chapter.Number != tt.number || tt.chapter.Title != tt.title {
			t.Errorf("wanted %v %q, got %v %q", tt.number, tt.title, tt.chapter.Number, tt.chapter.Title)
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
