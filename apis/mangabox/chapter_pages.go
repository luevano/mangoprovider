package mangabox

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal/mangadata"
	mango "github.com/luevano/mangoprovider"
)

var (
	cdnsRegex          = regexp.MustCompile(`cdns\s*=\s*\[([^\]]+)\]`)
	backupImageRegex   = regexp.MustCompile(`backupImage\s*=\s*\[([^\]]+)\]`)
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
	// every image is on each cdn: the first one is the url, the others fallbacks
	var urls [][]string
	cdns := uniqueCDNs(append(jsArray(body, cdnsRegex), jsArray(body, backupImageRegex)...))
	if images := jsArray(body, chapterImagesRegex); len(cdns) > 0 && len(images) > 0 {
		for _, image := range images {
			var onCDNs []string
			for _, cdn := range cdns {
				onCDNs = append(onCDNs, cdn+"/"+strings.TrimPrefix(image, "/"))
			}
			urls = append(urls, onCDNs)
		}
	} else {
		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		doc.Find("div.container-chapter-reader > img").Each(func(_ int, s *goquery.Selection) {
			if src := s.AttrOr("src", ""); src != "" {
				urls = append(urls, onOtherCDNs(src, cdns))
			}
		})
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("chapter %q has no pages", chapter.ID)
	}
	var pages []mangadata.Page
	for _, u := range urls {
		ext, err := mango.ImageExtension(u[0])
		if err != nil {
			return nil, err
		}
		pages = append(pages, &mango.Page{
			Ext:       ext,
			URL:       u[0],
			Fallbacks: u[1:],
			Headers:   map[string]string{"Referer": site, "User-Agent": m.userAgent},
			Chapter_:  &chapter,
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

// uniqueCDNs drops the trailing slashes and repeated cdns.
func uniqueCDNs(cdns []string) []string {
	var unique []string
	for _, cdn := range cdns {
		if cdn = strings.TrimSuffix(cdn, "/"); !slices.Contains(unique, cdn) {
			unique = append(unique, cdn)
		}
	}
	return unique
}

// onOtherCDNs: the image at src, then on the cdns that aren't its own (same path).
func onOtherCDNs(src string, cdns []string) []string {
	urls := []string{src}
	u, err := url.Parse(src)
	if err != nil || u.Host == "" {
		return urls
	}
	for _, cdn := range cdns {
		c, err := url.Parse(cdn)
		if err != nil || c.Host == "" || c.Host == u.Host {
			continue
		}
		urls = append(urls, c.Scheme+"://"+c.Host+u.EscapedPath())
	}
	return urls
}
