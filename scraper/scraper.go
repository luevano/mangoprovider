package scraper

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/headless"
)

var (
	chapterNumberRegex = regexp.MustCompile(`(?m)(\d+\.\d+|\d+)`)
	newLineCharacters  = regexp.MustCompile(`\r?\n`)
)

// Scraper: Generic scraper downloads html pages and parses them.
type Scraper struct {
	options *Options

	mangasCollector   *colly.Collector
	volumesCollector  *colly.Collector
	chaptersCollector *colly.Collector
	pagesCollector    *colly.Collector
}

// NewScraper: generates a new scraper with given configuration.
func NewScraper(options *Options, headlessOptions headless.Options) (*Scraper, error) {
	s := &Scraper{
		options: options,
	}

	collectorOptions := []colly.CollectorOption{
		colly.AllowURLRevisit(),
		colly.Async(true),
	}

	err := checkForRedirect(options)
	if err != nil {
		return nil, err
	}

	baseCollector := colly.NewCollector(collectorOptions...)
	baseCollector.SetRequestTimeout(30 * time.Second)

	if options.NeedsHeadlessBrowser {
		transport := headless.GetTransport(headlessOptions)
		baseCollector.WithTransport(transport)
	}

	err = s.setMangasCollector(baseCollector.Clone())
	if err != nil {
		return nil, err
	}

	err = s.setVolumesCollector(baseCollector.Clone())
	if err != nil {
		return nil, err
	}

	err = s.setChaptersCollector(baseCollector.Clone())
	if err != nil {
		return nil, err
	}

	err = s.setPagesCollector(baseCollector.Clone())
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Scraper) setMangasCollector(collector *colly.Collector) error {
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		elements := e.DOM.Find(s.options.MangaExtractor.Selector)
		mangas := e.Request.Ctx.GetAny("mangas").(*[]libmangal.Manga)

		elements.Each(func(_ int, selection *goquery.Selection) {
			link := s.options.MangaExtractor.URL(selection)
			url := e.Request.AbsoluteURL(link)
			title := cleanName(s.options.MangaExtractor.Title(selection))
			m := mango.Manga{
				Title:         title,
				AnilistSearch: title,
				URL:           url,
				ID:            s.options.MangaExtractor.ID(url),
				Cover:         s.options.MangaExtractor.Cover(selection),
			}
			*mangas = append(*mangas, m)
		})
	})

	err := setupCollector(collector, "manga", *s.options)
	if err != nil {
		return err
	}

	s.mangasCollector = collector
	return nil
}

func (s *Scraper) setVolumesCollector(collector *colly.Collector) error {
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		elements := e.DOM.Find(s.options.VolumeExtractor.Selector)
		manga := e.Request.Ctx.GetAny("manga").(mango.Manga)
		volumes := e.Request.Ctx.GetAny("volumes").(*[]libmangal.Volume)

		elements.Each(func(_ int, selection *goquery.Selection) {
			v := mango.Volume{
				Number: s.options.VolumeExtractor.Number(selection),
				Manga_: &manga,
			}
			*volumes = append(*volumes, v)
		})
	})

	err := setupCollector(collector, "volume", *s.options)
	if err != nil {
		return err
	}

	s.volumesCollector = collector
	return nil
}

func (s *Scraper) setChaptersCollector(collector *colly.Collector) error {
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		elements := e.DOM.Find(s.options.ChapterExtractor.Selector)
		volume := e.Request.Ctx.GetAny("volume").(mango.Volume)
		chapters := e.Request.Ctx.GetAny("chapters").(*[]libmangal.Chapter)

		elements.Each(func(_ int, selection *goquery.Selection) {
			link := s.options.ChapterExtractor.URL(selection)
			url := e.Request.AbsoluteURL(link)
			title := cleanName(s.options.ChapterExtractor.Title(selection))

			match := chapterNumberRegex.FindString(title)
			chapterNumber := float32(e.Index)
			if match != "" {
				number, err := strconv.ParseFloat(match, 32)
				if err == nil {
					chapterNumber = float32(number)
				}
			}

			var chapterDate libmangal.Date
			if s.options.ChapterExtractor.Date != nil {
				chapterDate = s.options.ChapterExtractor.Date(selection)
			}

			var scanlationGroup string
			if s.options.ChapterExtractor.ScanlationGroup != nil {
				scanlationGroup = s.options.ChapterExtractor.ScanlationGroup(selection)
			}

			c := mango.Chapter{
				Title:           title,
				ID:              s.options.ChapterExtractor.ID(url),
				URL:             url,
				Number:          chapterNumber,
				Date:            chapterDate,
				ScanlationGroup: scanlationGroup,
				Volume_:         &volume,
			}
			*chapters = append(*chapters, c)
		})
	})

	err := setupCollector(collector, "chapter", *s.options)
	if err != nil {
		return err
	}

	s.chaptersCollector = collector
	return nil
}

func (s *Scraper) setPagesCollector(collector *colly.Collector) error {
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		elements := e.DOM.Find(s.options.PageExtractor.Selector)
		chapter := e.Request.Ctx.GetAny("chapter").(mango.Chapter)
		pages := e.Request.Ctx.GetAny("pages").(*[]libmangal.Page)

		elements.Each(func(_ int, selection *goquery.Selection) {
			link := s.options.PageExtractor.URL(selection)
			ext := filepath.Ext(link)
			// remove some query params from the extension
			ext = strings.Split(ext, "?")[0]

			headers := map[string]string{
				"Referer":    chapter.URL,
				"Accept":     "image/webp,image/apng,image/*,*/*;q=0.8",
				"User-Agent": mango.UserAgent,
			}

			p := mango.Page{
				Extension: ext,
				URL:       link,
				Headers:   headers,
				Chapter_:  &chapter,
			}
			*pages = append(*pages, p)
		})
	})

	err := setupCollector(collector, "page", *s.options)
	if err != nil {
		return err
	}

	s.pagesCollector = collector
	return nil
}
