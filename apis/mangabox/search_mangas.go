package mangabox

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	mango "github.com/luevano/mangoprovider"
)

// SearchMangas uses the site's search (first page, 20 results: it goes through FlareSolverr).
func (m *mangabox) SearchMangas(ctx context.Context, store mango.Store, query string) ([]mangadata.Manga, error) {
	var mangas []mangadata.Manga
	if found, err := store.GetMangas(query, query, &mangas); err != nil || found {
		return mangas, err
	}
	mangas, err := m.search(ctx, store, query)
	if err != nil {
		return nil, err
	}
	return mangas, store.SetMangas(query, mangas)
}

func (m *mangabox) search(ctx context.Context, store mango.Store, query string) ([]mangadata.Manga, error) {
	body, _, err := m.get(ctx, store, "search/story/"+searchAlias(query))
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var mangas []mangadata.Manga
	doc.Find(".panel_story_list .story_item, div.list-truyen-item-wrap, div.list-comic-item-wrap").Each(func(_ int, s *goquery.Selection) {
		link := s.Find("h3 a").First()
		url, slug := link.AttrOr("href", ""), slugOf(link.AttrOr("href", ""))
		if slug == "" {
			return
		}
		mangas = append(mangas, m.toManga(slug, url, strings.TrimSpace(link.Text()), s.Find("img").First().AttrOr("src", ""), nil))
	})
	return mangas, nil
}

// MangaByID gets the manga by its slug (/manga/<slug>).
func (m *mangabox) MangaByID(ctx context.Context, store mango.Store, id string) (mangadata.Manga, bool, error) {
	manga, err := m.manga(ctx, store, url.PathEscape(id))
	if errors.Is(err, mango.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return manga, true, nil
}

// manga reads the manga's page, with more metadata than the search.
func (m *mangabox) manga(ctx context.Context, store mango.Store, slug string) (*mango.Manga, error) {
	body, site, err := m.get(ctx, store, "manga/"+slug)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	info := doc.Find("div.manga-info-top, div.panel-story-info").First()
	var details mangaDetails
	info.Find("li, td").Each(func(_ int, s *goquery.Selection) {
		label, value, ok := strings.Cut(s.Text(), ":")
		if !ok {
			return
		}
		switch strings.ToLower(strings.TrimSpace(label)) {
		case "author(s)", "author":
			for _, a := range strings.Split(value, ",") {
				if a = strings.TrimSpace(a); a != "" {
					details.authors = append(details.authors, a)
				}
			}
		case "status":
			details.status = strings.TrimSpace(value)
		}
	})
	info.Find("li.genres a, td:contains('Genres') + td a").Each(func(_ int, s *goquery.Selection) {
		details.genres = append(details.genres, strings.TrimSpace(s.Text()))
	})
	details.summary = strings.TrimSpace(doc.Find("div#noidungm, div#panel-story-info-description, div#contentBox").First().Text())
	for _, alt := range strings.Split(doc.Find(".story-alternative, tr:has(.info-alternative) h2").First().Text(), ";") {
		if alt = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(alt), "Alternative :")); alt != "" {
			details.alt = append(details.alt, alt)
		}
	}
	title := strings.TrimSpace(info.Find("h1, h2").First().Text())
	cover := doc.Find("div.manga-info-pic img, span.info-image img").First().AttrOr("src", "")
	return m.toManga(slug, site+"manga/"+slug, title, cover, &details), nil
}

type mangaDetails struct {
	authors, genres, alt []string
	status, summary      string
}

func (m *mangabox) toManga(slug, url, title, cover string, details *mangaDetails) *mango.Manga {
	meta := &mangadata.Metadata{
		EnglishTitle:      title,
		CoverImage:        cover,
		SourceURL:         url,
		ProviderID:        slug,
		ProviderIDCode:    "mbox",
		PublicationStatus: metadata.Status("UNKNOWN"),
	}
	if details != nil {
		meta.AuthorList = details.authors
		meta.GenreList = details.genres
		meta.Synonyms = details.alt
		meta.Summary = details.summary
		switch strings.ToLower(details.status) {
		case "ongoing":
			meta.PublicationStatus = metadata.StatusReleasing
		case "completed":
			meta.PublicationStatus = metadata.StatusFinished
		}
	}
	var md metadata.Metadata = meta
	return &mango.Manga{
		Title:     title,
		URL:       url,
		ID:        slug,
		Cover:     cover,
		Metadata_: &md,
	}
}

// slugOf the manga url, https://<site>/manga/<slug>
func slugOf(url string) string {
	_, slug, ok := strings.Cut(url, "/manga/")
	if !ok {
		return ""
	}
	slug, _, _ = strings.Cut(strings.Trim(slug, "/"), "/")
	return slug
}

var (
	aliasReplacer = strings.NewReplacer(
		"à", "a", "á", "a", "ạ", "a", "ả", "a", "ã", "a", "â", "a", "ầ", "a", "ấ", "a", "ậ", "a", "ẩ", "a", "ẫ", "a",
		"ă", "a", "ằ", "a", "ắ", "a", "ặ", "a", "ẳ", "a", "ẵ", "a",
		"è", "e", "é", "e", "ẹ", "e", "ẻ", "e", "ẽ", "e", "ê", "e", "ề", "e", "ế", "e", "ệ", "e", "ể", "e", "ễ", "e",
		"ì", "i", "í", "i", "ị", "i", "ỉ", "i", "ĩ", "i",
		"ò", "o", "ó", "o", "ọ", "o", "ỏ", "o", "õ", "o", "ô", "o", "ồ", "o", "ố", "o", "ộ", "o", "ổ", "o", "ỗ", "o",
		"ơ", "o", "ờ", "o", "ớ", "o", "ợ", "o", "ở", "o", "ỡ", "o",
		"ù", "u", "ú", "u", "ụ", "u", "ủ", "u", "ũ", "u", "ư", "u", "ừ", "u", "ứ", "u", "ự", "u", "ử", "u", "ữ", "u",
		"ỳ", "y", "ý", "y", "ỵ", "y", "ỷ", "y", "ỹ", "y", "đ", "d",
	)
	aliasPunct = regexp.MustCompile(`[!@%^*()+=<>?/,.:;' "&#\[\]~$_-]+`)
)

// searchAlias is the site's change_alias: the query as it goes in the search url.
func searchAlias(query string) string {
	s := aliasReplacer.Replace(strings.ToLower(query))
	return strings.Trim(aliasPunct.ReplaceAllString(s, "_"), "_")
}
