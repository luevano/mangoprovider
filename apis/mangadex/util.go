package mangadex

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	"github.com/luevano/mangodex"
	mango "github.com/luevano/mangoprovider"
)

const chapterNumberFormat = "Chapter %s"

// getChapterNum: The chapter's number, false for chapters without one (oneshots, extras).
func getChapterNum(chapter *mangodex.Chapter) (float32, bool, error) {
	chapterNumberStr := chapter.GetChapterNum()
	if chapterNumberStr == "-" {
		return 0, false, nil
	}

	chapterNumber, err := strconv.ParseFloat(chapterNumberStr, 32)
	if err != nil {
		return 0, false, err
	}
	return float32(chapterNumber), true, nil
}

// numberExtraChapters: Chapters without a number go right after the previous one (in the api's
// order), on a number no other chapter has; untitled ones are named after their number.
func numberExtraChapters(chapters []mangadata.Chapter, extras map[*mango.Chapter]bool) {
	taken := map[float32]bool{}
	for _, chapter := range chapters {
		if !extras[chapter.(*mango.Chapter)] {
			taken[chapter.Info().Number] = true
		}
	}
	var last float32
	for _, chapter := range chapters {
		c := chapter.(*mango.Chapter)
		if extras[c] {
			c.Number = mango.NextFreeExtraNumber(last, taken)
			taken[c.Number] = true
		}
		if c.Title == "" {
			c.Title = fmt.Sprintf(chapterNumberFormat, mango.FormattedFloat(c.Number))
		}
		last = c.Number
	}
}

// getDate: The published date, none if it doesn't parse.
func (d *dex) getDate(publishAt string) metadata.Date {
	publishedDate, err := time.Parse(time.RFC3339, publishAt)
	if err != nil {
		d.log.Log("chapter date %q: %s", publishAt, err)
		return metadata.Date{}
	}
	return mango.DateOf(publishedDate)
}

// Finds the first scanlator or the first username or "mangadex".
func (d *dex) getScanlator(relationships []*mangodex.Relationship) string {
	var scanlator string
	for _, relationship := range relationships {
		if relationship.Type == mangodex.RelationshipTypeScanlationGroup {
			groupRel, _ := relationship.Attributes.(*mangodex.ScanlationGroupAttributes)
			scanlator = groupRel.Name
			break
		}
	}
	// If no scanlator group is linked to the chapter, use the uploader user.
	if scanlator == "" {
		d.log.Log("no scanlator for chapter, using username")
		for _, relationship := range relationships {
			if relationship.Type == mangodex.RelationshipTypeUser {
				userRel, _ := relationship.Attributes.(*mangodex.UserAttributes)
				scanlator = userRel.Username
				break
			}
		}
	}
	// If even then the scanlator is not set, just use "mangadex".
	if scanlator == "" {
		scanlator = "MangaDex"
		d.log.Log("no scanlator or username for chapter, defaulting to %q", scanlator)
	}
	return scanlator
}

// chapterGroups: The ids and lowercased names of the chapter's scanlation groups.
func chapterGroups(relationships []*mangodex.Relationship) []string {
	var groups []string
	for _, relationship := range relationships {
		if relationship.Type != mangodex.RelationshipTypeScanlationGroup {
			continue
		}
		groups = append(groups, relationship.ID.String())
		if group, ok := relationship.Attributes.(*mangodex.ScanlationGroupAttributes); ok && group.Name != "" {
			groups = append(groups, strings.ToLower(group.Name))
		}
	}
	return groups
}

// matchGroup: The index of the first of wanted (names or ids, case insensitive) in groups,
// -1 if none is.
func matchGroup(groups, wanted []string) int {
	for i, w := range wanted {
		if slices.Contains(groups, strings.ToLower(strings.TrimSpace(w))) {
			return i
		}
	}
	return -1
}

// groupsKey: The group options for the chapters' cache id, empty without any.
func (d *dex) groupsKey() string {
	if len(d.options.ExcludeGroups) == 0 && len(d.options.PreferGroups) == 0 {
		return ""
	}
	return fmt.Sprintf("exclude:%s;prefer:%s", strings.Join(d.options.ExcludeGroups, ","), strings.Join(d.options.PreferGroups, ","))
}

// getUniqueChapters: One chapter per number: a downloadable one over an unavailable one, then
// the one from the first of prefer's groups, else from the group with the most uploads in the
// volume.
func getUniqueChapters(agg *aggregate, prefer []string) ([]mangadata.Chapter, error) {
	var chaptersFiltered []mangadata.Chapter
	for number, chapters := range agg.chaptersMap {
		switch len(chapters) {
		case 0:
			return nil, fmt.Errorf("remove duplicate chapters: no chapters for number %q", number)
		case 1:
			chaptersFiltered = append(chaptersFiltered, chapters[0])
		default:
			// prefer a downloadable upload over an unavailable one
			candidates := chapters
			var available []mangadata.Chapter
			for _, chapter := range chapters {
				if chapter.Info().Unavailable == "" {
					available = append(available, chapter)
				}
			}
			if len(available) != 0 {
				candidates = available
			}

			if preferred := preferredChapter(agg, candidates, prefer); preferred != nil {
				chaptersFiltered = append(chaptersFiltered, preferred)
				continue
			}

			var chapterTemp mangadata.Chapter
			maxUploads := 0
			for _, chapter := range candidates {
				scanlator := chapter.Info().ScanlationGroup
				// uploads is the total times the scanlator appears in the volume (how many uploads)
				uploads, found := agg.groupsCount[scanlator]
				if !found {
					return nil, fmt.Errorf("remove duplicate chapters: uploads of group %q not counted", scanlator)
				}
				// if it has appeared the most then select it as a candidate for the chapter #
				if uploads > maxUploads {
					chapterTemp = chapter
					maxUploads = uploads
				}
			}
			if chapterTemp == nil {
				return nil, fmt.Errorf("remove duplicate chapters: no candidate for number %q", number)
			}

			chaptersFiltered = append(chaptersFiltered, chapterTemp)
		}
	}
	return chaptersFiltered, nil
}

// preferredChapter: The candidate from the earliest group in prefer, nil if none is from them.
func preferredChapter(agg *aggregate, candidates []mangadata.Chapter, prefer []string) mangadata.Chapter {
	var best mangadata.Chapter
	bestRank := len(prefer)
	for _, chapter := range candidates {
		groups := agg.groups[chapter.(*mango.Chapter)]
		if groups == nil {
			groups = []string{strings.ToLower(chapter.Info().ScanlationGroup)}
		}
		if rank := matchGroup(groups, prefer); rank >= 0 && rank < bestRank {
			best, bestRank = chapter, rank
		}
	}
	return best
}
