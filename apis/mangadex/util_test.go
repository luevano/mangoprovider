package mangadex

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	"github.com/luevano/mangodex"
	mango "github.com/luevano/mangoprovider"
)

func TestGetChapterNum(t *testing.T) {
	str := func(s string) *string { return &s }
	tests := []struct {
		chapter  *string
		number   float32
		numbered bool
	}{
		{str("5"), 5, true},
		{str("12.5"), 12.5, true},
		{str("0"), 0, true},
		{nil, -1, false}, // no number (oneshots, extras)
		{str("10a"), 10, false},
		{str("10-a"), 10, false},
		{str("1,5"), 1, false},
		{str(""), -1, false},
		{str("NaN"), -1, false},
		{str("Inf"), -1, false},
		{str("extra"), -1, false},
	}
	for _, tt := range tests {
		chapter := &mangodex.Chapter{Attributes: mangodex.ChapterAttributes{Chapter: tt.chapter}}
		number, numbered := getChapterNum(chapter)
		if number != tt.number || numbered != tt.numbered {
			t.Errorf("%q: wanted %v, %v; got %v, %v", chapter.GetChapterNum(), tt.number, tt.numbered, number, numbered)
		}
	}
}

func TestNumberExtraChapters(t *testing.T) {
	one, extra, two := &mango.Chapter{Number: 1}, &mango.Chapter{Title: "Side story", Number: -1}, &mango.Chapter{Number: 2}
	extra2, extra3 := &mango.Chapter{Number: -1}, &mango.Chapter{Number: -1}
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

	chapters, err := getUniqueChapters(agg, nil)
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

func TestPreferGroups(t *testing.T) {
	chapter := func(group string) *mango.Chapter {
		return &mango.Chapter{ScanlationGroup: group}
	}
	big, small, fav := chapter("Big"), chapter("Small"), chapter("Fav")
	big2, small2 := chapter("Big"), chapter("Small")
	big3, other3 := chapter("Big"), chapter("Other")
	agg := &aggregate{
		chaptersMap: map[string][]mangadata.Chapter{
			"1": {big, small, fav}, // the first preferred group there wins
			"2": {big2, small2},    // groups not listed: by the chapter's group name
			"3": {other3, big3},    // none preferred: the most uploads
		},
		groupsCount: map[string]int{"Big": 9, "Small": 2, "Fav": 1, "Other": 1},
		groups: map[*mango.Chapter][]string{
			big: {"b-id", "big"}, small: {"s-id", "small"}, fav: {"f-id", "fav"},
		},
	}
	chapters, err := getUniqueChapters(agg, []string{"Missing", "F-ID", "small"})
	if err != nil {
		t.Fatal(err)
	}
	got := map[mangadata.Chapter]bool{}
	for _, c := range chapters {
		got[c] = true
	}
	if len(chapters) != 3 || !got[fav] || !got[small2] || !got[big3] {
		t.Errorf("wanted fav, small2 and big3, got %v", chapters)
	}
}

func TestChapterGroups(t *testing.T) {
	id := uuid.MustParse("6f0b3b9c-3e0b-4b8a-9f43-6c3c1f8f2a10")
	groups := chapterGroups([]*mangodex.Relationship{
		{ID: id, Type: mangodex.RelationshipTypeScanlationGroup, Attributes: &mangodex.ScanlationGroupAttributes{Name: "Some Scans"}},
		{ID: uuid.New(), Type: mangodex.RelationshipTypeUser, Attributes: &mangodex.UserAttributes{Username: "uploader"}},
	})
	if want := []string{id.String(), "some scans"}; !slices.Equal(groups, want) {
		t.Errorf("wanted %v, got %v", want, groups)
	}
	for _, tt := range []struct {
		wanted []string
		index  int
	}{
		{[]string{"other", " SOME scans "}, 1},
		{[]string{id.String()}, 0},
		{[]string{"uploader"}, -1},
		{nil, -1},
	} {
		if got := matchGroup(groups, tt.wanted); got != tt.index {
			t.Errorf("%q: wanted %d, got %d", tt.wanted, tt.index, got)
		}
	}
}

func TestGetDate(t *testing.T) {
	d := &dex{}
	if got := d.getDate("2024-03-05T10:00:00+00:00"); got.String() != "2024-03-05" {
		t.Errorf("wanted 2024-03-05, got %v", got)
	}
	// unknown stays unknown, not today
	if got := d.getDate(""); got.Year != 0 {
		t.Errorf("wanted no date, got %v", got)
	}
}

func TestMetadataLinks(t *testing.T) {
	d := &dex{}
	manga := &mangodex.Manga{ID: "a1c7c817-4e59-43b7-9365-09675a149a6f"}
	manga.Attributes.Title = mangodex.LocalisedStrings{Values: map[string]string{"en": "One Piece"}}
	year := 1997
	manga.Attributes.Year = &year
	status := mangodex.PublicationStatusOngoing
	manga.Attributes.Status = &status
	for _, tt := range []struct {
		mu   string
		want string
	}{
		{"pb8uwds", "[al:30013 mu:55099564912]"}, // the current base36 ids
		{"33", "[al:30013]"},                     // an old one, unknown to the api
	} {
		manga.Attributes.Links = mangodex.LocalisedStrings{Values: map[string]string{"al": "30013", "mu": tt.mu}}
		meta := d.dexToMetadata(manga)
		var got []string
		for _, id := range meta.ExtraIDs() {
			got = append(got, string(id.Code)+":"+id.Raw)
		}
		if s := "[" + strings.Join(got, " ") + "]"; s != tt.want {
			t.Errorf("mu %q: got %s, want %s", tt.mu, s, tt.want)
		}
		meta.AuthorList = []string{"ODA Eiichiro"} // from the relationships
		if err := metadata.Validate(meta); err != nil {
			t.Errorf("mu %q: %v", tt.mu, err)
		}
	}
}

func TestNumberExtraChaptersTaken(t *testing.T) {
	one, extra, half := &mango.Chapter{Number: 1}, &mango.Chapter{Title: "Extra", Number: -1}, &mango.Chapter{Number: 1.5}
	ten, tenA, two := &mango.Chapter{Number: 10}, &mango.Chapter{Title: "10a", Number: 10}, &mango.Chapter{Number: 2}
	numberExtraChapters([]mangadata.Chapter{one, extra, half, two, tenA, ten}, map[*mango.Chapter]bool{extra: true, tenA: true})
	if extra.Number != 1.6 {
		t.Errorf("wanted 1.6 (1.5 is taken), got %v", extra.Number)
	}
	// "10a" goes after 10, not after the previous chapter
	if tenA.Number != 10.5 {
		t.Errorf("wanted 10.5, got %v", tenA.Number)
	}
}
