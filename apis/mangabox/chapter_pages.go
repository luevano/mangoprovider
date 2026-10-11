package mangabox

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal/mangadata"
	mango "github.com/luevano/mangoprovider"
)

var (
	cdnsRegex          = regexp.MustCompile(`cdns\s*=\s*\[([^\]]+)\]`)
	chapterImagesRegex = regexp.MustCompile(`chapterImages\s*=\s*\[([^\]]+)\]`)
)

// ChapterPages: the chapter page has the image paths and the cdns they're on in a script,
// older chapters have the images in the reader.
func (m *mangabox) ChapterPages(ctx context.Context, store mango.Store, chapter mango.Chapter) ([]mangadata.Page, error) {
	slug := chapter.Volume_.Manga_.ID
	body, site, err := m.get(ctx, store, "manga/"+url.PathEscape(slug)+"/"+url.PathEscape(chapter.ID))
	if err != nil {
		return nil, err
	}
	var urls []string
	cdns, images := jsArray(body, cdnsRegex), jsArray(body, chapterImagesRegex)
	if len(cdns) > 0 && len(images) > 0 {
		// TODO: the other cdns (and backupImage) as fallbacks, like keiyoushi
		cdn := strings.TrimSuffix(cdns[0], "/")
		for _, image := range images {
			urls = append(urls, cdn+"/"+strings.TrimPrefix(image, "/"))
		}
	} else {
		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		doc.Find("div.container-chapter-reader > img").Each(func(_ int, s *goquery.Selection) {
			if src := s.AttrOr("src", ""); src != "" {
				urls = append(urls, src)
			}
		})
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("chapter %q has no pages", chapter.ID)
	}
	var pages []mangadata.Page
	for _, u := range urls {
		ext, err := mango.ImageExtension(u)
		if err != nil {
			return nil, err
		}
		pages = append(pages, &mango.Page{
			Ext:      ext,
			URL:      u,
			Headers:  map[string]string{"Referer": site, "User-Agent": m.userAgent},
			Chapter_: &chapter,
		})
	}
	return pages, nil
}

// jsArray reads a js array of strings ("a\/b", "c") matched by re.
func jsArray(body []byte, re *regexp.Regexp) []string {
	match := re.FindSubmatch(body)
	if match == nil {
		return nil
	}
	var items []string
	for _, item := range strings.Split(string(match[1]), ",") {
		item = strings.ReplaceAll(strings.Trim(strings.TrimSpace(item), `"'`), `\/`, "/")
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}
