package mangoprovider

import (
	"testing"

	"github.com/luevano/libmangal/metadata"
	"github.com/luevano/libmangal/netutil"
	"github.com/luevano/mangoplus/creators"
)

var matchingTests = []struct {
	original string
	vol      string
	chap     string
	part     string
	title    string
}{
	{"chapter 1.5: name", "", "1.5", "", "name"},
	{"vol. 1 chapter 1.5: name", "1", "1.5", "", "name"},
	{"vol. 1 chapter 1.5, part 2: name", "1", "1.5", "2", "name"},
	{"vol.22 chapter 1.5 name", "22", "1.5", "", "name"},
	{"CHAPTER 2 - NAME", "", "2", "", "NAME"},
	{"V.1.5 CHAPTER 2 - NAME", "1.5", "2", "", "NAME"},
	{"CHAPTER #2 NAME", "", "2", "", "NAME"},
	{"chapter 6: CHAPTER name", "", "6", "", "CHAPTER name"},
	{"flavor 21.6 something something", "", "21.6", "", "something something"},
	{"no. 1: chapter name", "", "1", "", "chapter name"},
	{"no. #1: chapter name", "", "1", "", "chapter name"},
	{"page 5: chapter NAME 123", "", "5", "", "chapter NAME 123"},
	{"page 5 _ name", "", "5", "", "name"},
	{"something6 name", "", "6", "", "name"},
	{"# 1.1 name", "", "1.1", "", "name"},
	{"#1 name", "", "1", "", "name"},
	{"1.1 - name", "", "1.1", "", "name"},
	{"10: test", "", "10", "", "test"},
	{"#14.1 hello", "", "14.1", "", "hello"},
	{"#14.50: test", "", "14.50", "", "test"},
	{"65-2: dashed chapter", "", "65-2", "", "dashed chapter"},
	{"Chapter 5, Part 1: test", "", "5", "1", "test"},
	{"Chapter 5: Part 1: test", "", "5", "1", "test"},
	{"Mission 5 Part 2 : test", "", "5", "2", "test"},
	{"Mission 5 Part 2-2 : test", "", "5", "2-2", "test"},
	{"Mission 5-2 Part 2-2 : test", "", "5-2", "2-2", "test"},
	{"Chapter 133, Part 2: Of One Cloth-Flutter", "", "133", "2", "Of One Cloth-Flutter"},
	{"Chapter 262: Inhuman Makyo Shinjuku Showdown, Part 34", "", "262", "", "Inhuman Makyo Shinjuku Showdown, Part 34"},
	{"Chapter 262-2: Inhuman Makyo Shinjuku Showdown, Part 34-2", "", "262-2", "", "Inhuman Makyo Shinjuku Showdown, Part 34-2"},
	{"Vol.3 Chapter 12 : Teresa Of The Faint Smile, Part 1", "3", "12", "", "Teresa Of The Faint Smile, Part 1"},
	{"Chapter 1 : The Silver-eyed Killer", "", "1", "", "The Silver-eyed Killer"},
	{"Chapter 11 : Darkness In Paradise 7", "", "11", "", "Darkness In Paradise 7"},
	{"Chapter 30 : The Slashers 6", "", "30", "", "The Slashers 6"},
}

func TestChapterNameRegexMatch(t *testing.T) {
	for _, tt := range matchingTests {
		matchGroups := ReNamedGroups(ChapterNameRegex, tt.original)
		title, ok := matchGroups[ChapterTitleID]
		if !ok {
			t.Errorf("No match (should match); for title %q", tt.original)
		} else if title != tt.title {
			t.Errorf("Wrong title match (should match); got: %q, wanted: %q", title, tt.title)
		} else {
			t.Logf("Title match (ok); got: %q for title %q", title, tt.original)
		}

		vol := matchGroups[VolumeNumberID]
		if vol != tt.vol {
			t.Errorf("Wrong volume match (should match); got: %q, wanted: %q; original: %q", vol, tt.vol, tt.original)
		}

		chap := matchGroups[ChapterNumberID]
		if chap != tt.chap {
			t.Errorf("Wrong chapter match (should match); got: %q, wanted: %q; original: %q", chap, tt.chap, tt.original)
		}

		part := matchGroups[PartNumberID]
		if part != tt.part {
			t.Errorf("Wrong part match (should match); got: %q, wanted: %q; original: %q", part, tt.part, tt.original)
		}
	}
}

func TestChapterNameRegexNoMatch(t *testing.T) {
	tests := []struct {
		original string
	}{
		{"Chapter 10"},
		{"MISSION 100"},
		{"chapter 123.5"},
		{"Chapter 123-5"},
		{"mission 5 part 2"},
		{"Mission 6, Part 6.5"},
		{"MISSION 10: Part 2"},
		{"MISSION 10: Part 2-2"},
		{"MISSION 10-2: Part 2-2"},
	}
	for _, tt := range tests {
		matchGroups := ReNamedGroups(ChapterNameRegex, tt.original)
		title, ok := matchGroups[ChapterTitleID]
		if ok {
			// When matching "Part X" it's fine, as there are no negative lookaheads
			// in golang regex, this is a separate check if its found
			if ChapterNameExcludeRegex.MatchString(title) {
				t.Logf("Part match (ok); got %q for title %q", title, tt.original)
			} else {
				t.Errorf("Match (shouldn't match); got %q for title %q", title, tt.original)
			}
		} else {
			t.Logf("No Match (ok); for title %q", tt.original)
		}
	}
}

func TestChapterNumberRegexMatch(t *testing.T) {
	tests := []struct {
		original string
		want     string
	}{
		{"Chapter 1: The Silver-eyed Killer", "1"},
		{"Chapter 11: Darkness In Paradise 7", "11"},
		{"Chapter 30: The Slashers 6", "30"},
		{"chapter 1.5: name", "1.5"},
		{"chapter 1.5 name with number 123-5", "1.5"},
		{"CHAPTER 2 - 2nd NAME", "2"},
		{"CHAPTER #2-5 NAME", "2-5"},
		{"chapter 6.2: CHAPTER name", "6.2"},
		{"Chapter 133, Part 2: Of One Cloth-Flutter", "133"},
		{"Mission 5 Part 2 : test", "5"},
		{"#5", "5"},
		{"5-1", "5-1"},
		{"61.45", "61.45"},
	}
	for _, tt := range tests {
		match := ChapterNumberRegex.FindString(tt.original)
		if match == "" {
			t.Errorf("No match (should match); for title %q, wanted %q", tt.original, tt.want)
		} else if match != tt.want {
			t.Errorf("Wrong match (should match); got: %q, wanted: %q", match, tt.want)
		} else {
			t.Logf("Match (ok); got: %q for title %q", match, tt.original)
		}
	}
}

func TestParseChapterNumber(t *testing.T) {
	tests := []struct {
		title  string
		number float32
		ok     bool
	}{
		{"Chapter 5: The start", 5, true},
		{"Vol.2 Chapter 12.5 - Side story", 12.5, true},
		{"#012-2", 12.2, true},
		{"#001", 1, true},
		{"Extra", 0, false},
		{"Prologue", 0, false},
		// a volume prefix followed by an explicit chapter
		{"Vol. 2 Chapter 10", 10, true},
		{"Vol.2 Ch.10", 10, true},
		{"v2 c10", 10, true},
		{"Vol 1 Chapter 3", 3, true},
		{"Season 2 Episode 10", 10, true},
		{"S2 Ep 5", 5, true},
		{"Tome 2 Chapitre 10", 10, true},
		{"Volume 2 - Chapter 7", 7, true},
		{"Vol. 2 #5", 5, true},
		{"Tomo 1 Capítulo 4", 4, true},
		{"Band 3 Kapitel 12", 12, true},
		{"第2巻 第10話", 10, true},
		{"2巻 第10话", 10, true},
		{"3권 第7화", 7, true},
		// unchanged
		{"Chapter 2 something something 5", 2, true},
		{"Chapter 2 - 5 Centimeters", 2, true},
		{"Vol. 10 Chapter 2 - 100 Days", 2, true},
		{"Vol.2 Ch.10 - Title", 10, true},
		{"Volume 3", 3, true},
		{"Vol.3", 3, true},
		{"Vol. 2 Extra 5", 2, true},
		{"Book 3: The End", 3, true},
		{"Vol.1 Part 2", 1, true},
		{"Mission 5 Part 2", 5, true},
		{"#004-5", 4.5, true},
		{"第10話", 10, true},
		{"Episode 10", 10, true},
		{"Chapter 201 - Epilogue", 201, true},
		{"Chapter 10a", 10, true},
		{"10", 10, true},
	}
	for _, tt := range tests {
		number, ok := ParseChapterNumber(tt.title)
		if number != tt.number || ok != tt.ok {
			t.Errorf("%q: wanted %v, %v; got %v, %v", tt.title, tt.number, tt.ok, number, ok)
		}
	}
}

func TestNextExtraNumber(t *testing.T) {
	tests := map[float32]float32{0: 0.5, 5: 5.5, 5.5: 5.6, 5.6: 5.7, 5.9: 5.91, 5.95: 5.96, 5.99: 5.991}
	for last, want := range tests {
		if got := NextExtraNumber(last); got != want {
			t.Errorf("after %v: wanted %v, got %v", last, want, got)
		}
	}
}

func TestNextFreeExtraNumber(t *testing.T) {
	taken := func(numbers ...float32) map[float32]bool {
		m := map[float32]bool{}
		for _, n := range numbers {
			m[n] = true
		}
		return m
	}
	tests := []struct {
		last  float32
		taken map[float32]bool
		want  string
	}{
		{5, nil, "5.5"},
		{5, taken(5.5), "5.6"},
		{10.5, taken(10.6), "10.7"},
		{5, taken(5, 5.5, 5.6, 5.7, 5.8, 5.9, 6), "5.91"},
		{5.9, taken(5.91, 5.92, 6), "5.93"},
		{5.99, taken(6), "5.991"},
		{1234, taken(1234.5), "1234.6"},
	}
	for _, tt := range tests {
		got := NextFreeExtraNumber(tt.last, tt.taken)
		if FormattedFloat(got) != tt.want || tt.taken[got] {
			t.Errorf("after %v (taken %v): wanted %s, got %s", tt.last, tt.taken, tt.want, FormattedFloat(got))
		}
	}
}

func TestImageExtension(t *testing.T) {
	tests := map[string]string{
		"https://host/a/1.png":         ".png",
		"https://host/a/1.jpg?v=1.2":   ".jpg",
		"1-abc123.webp":                ".webp",
		"https://host/a/1.jpeg#ab12cd": ".jpeg",
	}
	for in, want := range tests {
		if got, err := ImageExtension(in); err != nil || got != want {
			t.Errorf("%q: wanted %q, got %q, %v", in, want, got, err)
		}
	}
	if _, err := ImageExtension("https://host/a/page"); err == nil {
		t.Error("an url without an extension should be an error")
	}
}

func TestEscapePath(t *testing.T) {
	if got := EscapePath("1/one piece?x#y"); got != "1/one%20piece%3Fx%23y" {
		t.Errorf("got %q", got)
	}
}

func TestParseStatus(t *testing.T) {
	for status, want := range map[string]metadata.Status{
		"Ongoing":     metadata.StatusReleasing,
		"COMPLETED":   metadata.StatusFinished,
		"Hiatus":      metadata.StatusHiatus,
		"Dropped":     metadata.StatusCancelled,
		"cancelled":   metadata.StatusCancelled,
		"Coming Soon": metadata.StatusNotYetReleased,
		"Season End":  metadata.Status("UNKNOWN"),
		"publishing":  metadata.StatusReleasing,
		"Complete":    metadata.StatusFinished,
		"finished":    metadata.StatusFinished,
		"Canceled":    metadata.StatusCancelled,
	} {
		if got := ParseStatus(status); got != want {
			t.Errorf("%q: wanted %q, got %q", status, want, got)
		}
	}
}

func TestCountryOfType(t *testing.T) {
	for typ, want := range map[string]string{"Manga": "JP", " manhwa ": "KR", "MANHUA": "CN", "Webtoon": ""} {
		if got := CountryOfType(typ); got != want {
			t.Errorf("%q: wanted %q, got %q", typ, want, got)
		}
	}
}

func TestDefaultUserAgents(t *testing.T) {
	// mangoplus doesn't import libmangal, its creators default is a copy
	if DefaultOptions().UserAgent != netutil.BrowserUserAgent || creators.DefaultOptions().UserAgent != netutil.BrowserUserAgent {
		t.Errorf("default user agents %q, %q; want netutil.BrowserUserAgent", DefaultOptions().UserAgent, creators.DefaultOptions().UserAgent)
	}
}

func TestNextFreeExtraNumberLongRun(t *testing.T) {
	// 40 extras in a row stay distinct (20 past 10000, float32), later ones never take the next
	// whole number
	for start, fit := range map[float32]int{81: 40, 12345: 20} {
		taken := map[float32]bool{start: true, start + 1: true}
		last := start
		for i := range 300 {
			n := NextFreeExtraNumber(last, taken)
			if n <= start || n >= start+1 || (i < fit && taken[n]) {
				t.Fatalf("after %v, extra %d: %v", start, i, n)
			}
			taken[n], last = true, n
		}
	}
}

func TestNextFreeExtraNumberPastFreeWholeNumbers(t *testing.T) {
	// with no chapter after it, a long run (mangapill's "Group 2" chapters) goes on past the
	// whole numbers without taking them
	start := float32(77)
	taken := map[float32]bool{start: true}
	last := start
	for i := range 300 {
		n := NextFreeExtraNumber(last, taken)
		if taken[n] || n <= last || n == float32(int(n)) {
			t.Fatalf("extra %d after %v: %v", i, last, n)
		}
		taken[n], last = true, n
	}
}
