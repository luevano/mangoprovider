package scraper

import (
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/luevano/libmangal"
	"github.com/luevano/mangoprovider/mango"
)

// TODO: ADD VOLUME COLLECTOR

var (
	chapterNumberRegex = regexp.MustCompile(`(?m)(\d+\.\d+|\d+)`)
	newLineCharacters  = regexp.MustCompile(`\r?\n`)
)

// Scraper: Generic scraper downloads html pages and parses them.
type Scraper struct {
	mangasCollector   *colly.Collector
	volumesCollector  *colly.Collector
	chaptersCollector *colly.Collector
	pagesCollector    *colly.Collector

	// TODO: Need to provide store gokv.Store solution
	// Previously "github.com/belphemur/mangal/provider/cacher"
	// and "github.com/belphemur/mangal/source"
	// cache struct {
	// 	mangas   *cacher.Cacher[[]*source.Manga]   `json:"mangas,omitempty"`
	// 	chapters *cacher.Cacher[[]*source.Chapter] `json:"chapters,omitempty"`
	// }

	options *Options
}

// NewScraper: generates a new scraper with given configuration.
func NewScraper(conf *Options) (*Scraper, error){
	s := Scraper{
		options: conf,
	}
	// TODO: use store (gokv.Store) for cache
	// "github.com/belphemur/mangal/provider/cacher"
	// s.cache.mangas = cacher.NewCacher[[]*source.Manga](fmt.Sprintf("%s_%s", conf.Name, "mangas"), 6*time.Hour)
	// s.cache.chapters = cacher.NewCacher[[]*source.Chapter](fmt.Sprintf("%s_%s", conf.Name, "chapters"), 6*time.Hour)

	collectorOptions := []colly.CollectorOption{
		colly.AllowURLRevisit(),
		colly.Async(true),
	}

	err := checkForRedirect(conf)
	if err != nil {
		panic(err)
	}

	baseCollector := colly.NewCollector(collectorOptions...)
	baseCollector.SetRequestTimeout(30 * time.Second)
	// TODO: add headless
	// "github.com/belphemur/mangal/provider/generic/headless"
	// if conf.NeedsHeadlessBrowser {
	// 	transport := headless.GetTransportSingleton()
	// 	baseCollector.WithTransport(transport)
	// }

	mangasCollector := baseCollector.Clone()
	mangasCollector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Referer", "https://google.com")
		r.Headers.Set("accept-language", "en-US")
		r.Headers.Set("Accept", "text/html")
		r.Headers.Set("Host", s.options.BaseURL)
		r.Headers.Set("User-Agent", mango.UserAgent)
	})

	// Get mangas
	mangasCollector.OnHTML("html", func(e *colly.HTMLElement) {
		elements := e.DOM.Find(s.options.MangaExtractor.Selector)
		mangas := e.Request.Ctx.GetAny("mangas").(*[]libmangal.Manga)

		elements.Each(func(_ int, selection *goquery.Selection) {
			link := s.options.MangaExtractor.URL(selection)
			url := e.Request.AbsoluteURL(link)
			title := cleanName(s.options.MangaExtractor.Name(selection))

			// TODO: need to check ID creation,
			// still not sure what Manga.Banner is supposed to be
			m := mango.Manga{
				Title:         title,
				AnilistSearch: title,
				URL:           url,
				ID:            filepath.Base(url),
				Cover:         s.options.MangaExtractor.Cover(selection),
			}

			*mangas = append(*mangas, &m)
		})
	})

	err = mangasCollector.Limit(&colly.LimitRule{
		Parallelism: int(s.options.Parallelism),
		RandomDelay: s.options.Delay,
		DomainGlob:  "*",
	})
	if err != nil {
		return nil, err
	}

	// TODO: ADD VOLUME COLLECTOR

	// chaptersCollector := baseCollector.Clone()
	// chaptersCollector.OnRequest(func(r *colly.Request) {
	// 	r.Headers.Set("Referer", r.Ctx.GetAny("manga").(*mango.Manga).URL)
	// 	r.Headers.Set("accept-language", "en-US")
	// 	r.Headers.Set("Accept", "text/html")
	// 	r.Headers.Set("Host", s.config.BaseURL)
	// 	r.Headers.Set("User-Agent", mango.UserAgent)
	// })

	// // Get chapters
	// chaptersCollector.OnHTML("html", func(e *colly.HTMLElement) {
	// 	elements := e.DOM.Find(s.config.ChapterExtractor.Selector)
	// 	manga := e.Request.Ctx.GetAny("manga").(*mango.Manga)

	// 	elements.Each(func(i int, selection *goquery.Selection) {
	// 		link := s.config.ChapterExtractor.URL(selection)
	// 		url := e.Request.AbsoluteURL(link)
	// 		title := cleanName(s.config.ChapterExtractor.Name(selection))

	// 		match := chapterNumberRegex.FindString(title)
	// 		chapterNumber := float32(e.Index)
	// 		if match != "" {
	// 			number, err := strconv.ParseFloat(match, 32)
	// 			if err == nil {
	// 				chapterNumber = float32(number)
	// 			}
	// 		}

	// 		// TODO: enable this once the chapter date is implemented
	// 		// var chapterDate *time.Time
	// 		// if s.config.ChapterExtractor.Date != nil {
	// 		// 	chapterDate = s.config.ChapterExtractor.Date(selection)
	// 		// }

	// 		c := mango.Chapter{
	// 			Title:   title,
	// 			ID:      filepath.Base(url),
	// 			URL:     url,
	// 			Number:  chapterNumber,
	// 			Volume_: &mango.Volume{}, // TODO: add the actual volume once it's implemented
	// 			// in old mangal, Volume was a string
	// 			// Volume: s.config.ChapterExtractor.Volume(selection),
	// 			// TODO: add chapter date once it's implemented in libmangal
	// 			// ChapterDate: chapterDate,
	// 		}

	// 		manga.Chapters = append(manga.Chapters, &c)
	// 	})
	// })
	// _ = chaptersCollector.Limit(&colly.LimitRule{
	// 	Parallelism: int(s.config.Parallelism),
	// 	RandomDelay: s.config.Delay,
	// 	DomainGlob:  "*",
	// })

	// pagesCollector := baseCollector.Clone()
	// pagesCollector.OnRequest(func(r *colly.Request) {
	// 	r.Headers.Set("Referer", r.Ctx.GetAny("chapter").(*source.Chapter).URL)
	// 	r.Headers.Set("accept-language", "en-US")
	// 	r.Headers.Set("Accept", "text/html")
	// 	r.Headers.Set("User-Agent", mango.UserAgent)
	// })

	// // Get pages
	// pagesCollector.OnHTML("html", func(e *colly.HTMLElement) {
	// 	elements := e.DOM.Find(s.config.PageExtractor.Selector)
	// 	chapter := e.Request.Ctx.GetAny("chapter").(*source.Chapter)

	// 	elements.Each(func(i int, selection *goquery.Selection) {
	// 		link := s.config.PageExtractor.URL(selection)
	// 		ext := filepath.Ext(link)
	// 		// remove some query params from the extension
	// 		ext = strings.Split(ext, "?")[0]

	// 		page := source.Page{
	// 			URL:       link,
	// 			Index:     uint16(i),
	// 			Chapter:   chapter,
	// 			Extension: ext,
	// 		}
	// 		chapter.Pages = append(chapter.Pages, &page)
	// 	})
	// })
	// _ = pagesCollector.Limit(&colly.LimitRule{
	// 	Parallelism: int(s.config.Parallelism),
	// 	RandomDelay: s.config.Delay,
	// 	DomainGlob:  "*",
	// })

	s.mangasCollector = mangasCollector
	// s.chaptersCollector = chaptersCollector
	// s.pagesCollector = pagesCollector

	return &s, nil
}

func checkForRedirect(conf *Options) error {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(conf.BaseURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	defer client.CloseIdleConnections()

	if resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusFound {
		loc, err := resp.Location()
		if err != nil {
			return err
		}
		conf.BaseURL = loc.String()
	}
	return nil
}

func standardizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func cleanName(name string) string {
	return standardizeSpaces(newLineCharacters.ReplaceAllString(name, " "))
}
