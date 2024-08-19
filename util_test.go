package mangoprovider

import "testing"

func TestChapterNameRegexMatch(t *testing.T) {
	tests := []struct {
		original string
		want     string
	}{
		{"chapter 1.5: name", "name"},
		{"vol. 1 chapter 1.5: name", "name"},
		{"chapter 1.5 name", "name"},
		{"vol.22 chapter 1.5 name", "name"},
		{"CHAPTER 2 - NAME", "NAME"},
		{"V.1.5 CHAPTER 2 - NAME", "NAME"},
		{"CHAPTER #2 NAME", "NAME"},
		{"chapter 6: CHAPTER name", "CHAPTER name"},
		{"flavor 21.6 something something", "something something"},
		{"no. 1: chapter name", "chapter name"},
		{"no. #1: chapter name", "chapter name"},
		{"page 5: chapter NAME 123", "chapter NAME 123"},
		{"page 5 _ name", "name"},
		{"something6 name", "name"},
		{"# 1.1 name", "name"},
		{"#1 name", "name"},
		{"1.1 - name", "name"},
		{"10: test", "test"},
		{"#14.1 hello", "hello"},
		{"#14.5: test", "test"},
		{"65-2: dashed chapter", "dashed chapter"},
		{"Chapter 5, Part 1: test", "test"},
		{"Chapter 5: Part 1: test", "test"},
		{"Mission 5 Part 2 : test", "test"},
		{"Mission 5 Part 2-2 : test", "test"},
		{"Mission 5-2 Part 2-2 : test", "test"},
		{"Chapter 133, Part 2: Of One Cloth-Flutter", "Of One Cloth-Flutter"},
		{"Chapter 262: Inhuman Makyo Shinjuku Showdown, Part 34", "Inhuman Makyo Shinjuku Showdown, Part 34"},
		{"Chapter 262-2: Inhuman Makyo Shinjuku Showdown, Part 34-2", "Inhuman Makyo Shinjuku Showdown, Part 34-2"},
		{"Vol.3 Chapter 12 : Teresa Of The Faint Smile, Part 1", "Teresa Of The Faint Smile, Part 1"},
	}
	for _, tt := range tests {
		matchGroups := ReNamedGroups(ChapterNameRegex, tt.original)
		title, ok := matchGroups[ChapterTitleID]
		if !ok {
			t.Errorf("No match (should match); for title %q", tt.original)
		} else if title != tt.want {
			t.Errorf("Wrong match (should match); got: %q, wanted: %q", title, tt.want)
		} else {
			t.Logf("Match (ok); got: %q for title %q", title, tt.original)
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
