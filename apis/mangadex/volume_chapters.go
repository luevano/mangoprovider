package mangadex

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/mangodex"
	mango "github.com/luevano/mangoprovider"
)

// maxResults: MangaDex refuses an offset+limit over 10000.
var maxResults = 10000

// Contains the actual chapter list as well as helper values for filtering.
type aggregate struct {
	chapters    []mangadata.Chapter
	chaptersMap map[string][]mangadata.Chapter
	groupsCount map[string]int
	// extras: chapters without a number
	extras map[*mango.Chapter]bool
	// groups: the chapters' scanlation groups (ids and lowercased names)
	groups map[*mango.Chapter][]string
}

func (d *dex) VolumeChapters(ctx context.Context, store mango.Store, volume mango.Volume) ([]mangadata.Chapter, error) {
	limit := 100
	agg := aggregate{
		chapters:    []mangadata.Chapter{},
		chaptersMap: map[string][]mangadata.Chapter{},
		groupsCount: map[string]int{},
		extras:      map[*mango.Chapter]bool{},
		groups:      map[*mango.Chapter][]string{},
	}

	volumeNumber := volume.Label
	// volumes cached before Label existed
	if volumeNumber == "" {
		volumeNumber = volume.String()
		if volume.None {
			volumeNumber = "none"
		}
	}

	params := url.Values{}
	params.Set("manga", volume.Manga_.ID)
	params.Add("volume[]", volumeNumber)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("order[chapter]", mangodex.OrderAscending)
	// none is any language
	if d.filter.Language != "" {
		params.Add("translatedLanguage[]", d.filter.Language)
	}
	params.Add("includes[]", string(mangodex.RelationshipTypeScanlationGroup))
	params.Add("includes[]", string(mangodex.RelationshipTypeUser))
	if !d.filter.ShowUnavailableChapters {
		params.Set("includeEmptyPages", "0")
	}
	if d.filter.ShowRetiredChapters {
		params.Set("includeUnavailable", "1")
	}

	ratings := []mangodex.ContentRating{mangodex.ContentRatingSafe, mangodex.ContentRatingSuggestive}
	if d.filter.NSFW {
		ratings = append(ratings, mangodex.ContentRatingPorn)
		ratings = append(ratings, mangodex.ContentRatingErotica)
	}
	for _, rating := range ratings {
		params.Add("contentRating[]", string(rating))
	}

	// This doesn't include the offset so that it retrieves and saves all the chapters.
	cacheID := fmt.Sprintf("%s-%s", params.Encode(), d.filter.String())
	if groups := d.groupsKey(); groups != "" {
		cacheID += "-" + groups
	}
	var chapters []mangadata.Chapter
	found, err := store.GetChapters(cacheID, volume, &chapters)
	if err != nil {
		return nil, err
	}
	if found {
		return chapters, nil
	}

	offset := 0
	for page := 1; ; page++ {
		if page > 1 {
			d.log.Progress(ctx, 0, 0, "volume %s: %d chapters so far, getting page %d", volume.String(), len(agg.chapters), page)
		}
		// The offset is set on each iteration, shouldn't be included in the cacheID.
		params.Set("offset", strconv.Itoa(offset))
		ended, err := d.populateChapters(ctx, &agg, params, volume)
		if err != nil {
			return nil, err
		}
		if ended {
			break
		}
		offset += limit
		if offset+limit > maxResults {
			d.log.Log("volume %s: list cut at %d chapters, MangaDex lists at most %d", volume.String(), len(agg.chapters), maxResults)
			break
		}
	}
	numberExtraChapters(agg.chapters, agg.extras)

	var chaptersFiltered []mangadata.Chapter

	if d.filter.AvoidDuplicateChapters {
		chaptersFiltered, err = getUniqueChapters(&agg, d.options.PreferGroups)
		if err != nil {
			return nil, err
		}
	} else {
		chaptersFiltered = agg.chapters
	}

	sort.SliceStable(chaptersFiltered, func(i, j int) bool {
		return chaptersFiltered[i].Info().Number < chaptersFiltered[j].Info().Number
	})

	err = store.SetChapters(cacheID, chaptersFiltered)
	if err != nil {
		return nil, err
	}

	return chaptersFiltered, nil
}

// Make the request and parse the responses, populating the actual chapter list and extra info useful for filtering.
func (d *dex) populateChapters(ctx context.Context, agg *aggregate, params url.Values, volume mango.Volume) (bool, error) {
	chapterList, err := d.client.Chapter.ListContext(ctx, params)
	if err != nil {
		return false, err
	}

	if len(chapterList) == 0 {
		return true, nil
	}

	for _, chapter := range chapterList {
		// Retired, external and empty chapters can't be downloaded, only shown if wanted.
		// retired ones are only sent when asked for, so they're always kept.
		var unavailable string
		switch {
		case chapter.Attributes.IsUnavailable:
			unavailable = "retired"
		case chapter.Attributes.ExternalURL != nil:
			unavailable = "external"
		case chapter.Attributes.Pages == 0:
			unavailable = "0 pages"
		}
		if unavailable != "" && !chapter.Attributes.IsUnavailable && !d.filter.ShowUnavailableChapters {
			continue
		}

		chapterNumber, numbered := getChapterNum(chapter)

		chapterID := chapter.ID
		groups := chapterGroups(chapter.Relationships)
		if excluded := matchGroup(groups, d.options.ExcludeGroups); excluded >= 0 {
			d.log.Log("chapter %s: group %q excluded", chapterID, d.options.ExcludeGroups[excluded])
			continue
		}
		date := d.getDate(chapter.Attributes.PublishAt)
		scanlator := d.getScanlator(chapter.Relationships)

		c := mango.Chapter{
			Title:           chapter.GetTitle(),
			ID:              chapterID,
			URL:             fmt.Sprintf("%schapter/%s", website, chapterID),
			Number:          chapterNumber,
			Date:            date,
			ScanlationGroup: scanlator,
			Unavailable:     unavailable,
			Volume_:         &volume,
		}

		// chapters without a number (oneshots, extras) aren't duplicates of each other
		mapKey := mango.FormattedFloat(chapterNumber)
		if !numbered {
			mapKey = chapterID
			agg.extras[&c] = true
		}
		agg.groups[&c] = groups
		agg.chaptersMap[mapKey] = append(agg.chaptersMap[mapKey], &c)
		agg.groupsCount[scanlator] += 1
		agg.chapters = append(agg.chapters, &c)
	}

	// If received 100 entries means it probably has more.
	if len(chapterList) == 100 {
		return false, nil
	}

	return true, nil
}
