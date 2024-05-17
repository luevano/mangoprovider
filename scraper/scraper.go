package scraper

import (
	"fmt"
	"regexp"
	"time"

	"github.com/gocolly/colly/v2"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/headless"
)

var (
	chapterNumberRegex = regexp.MustCompile(`(?m)(\d+\.\d+|\d+)`)
	newLineCharacters  = regexp.MustCompile(`\r?\n`)
)

// Scraper: Generic scraper downloads html pages and parses them.
type Scraper struct {
	config    *Configuration
	options   mango.Options
	collector *colly.Collector
}

// NewScraper: generates a new scraper with given configuration and options.
func NewScraper(config *Configuration, options mango.Options) (scraper *Scraper, err error) {
	// Set the parallelism if not set by the scraper, use the provided parallelism option
	if config.Parallelism == 0 {
		mango.Log(fmt.Sprintf("setting parallelism to %d", options.Parallelism))
		config.Parallelism = options.Parallelism
	}
	// Set the new BaseURL if there are redirects.
	err = setBaseURLOnRedirect(config)
	if err != nil {
		return nil, err
	}

	scraper = &Scraper{
		config:  config,
		options: options,
	}
	err = scraper.setCollector()
	if err != nil {
		return nil, err
	}

	return scraper, nil
}

func (s *Scraper) setCollector() error {
	collectorOptions := []colly.CollectorOption{
		colly.AllowURLRevisit(),
		colly.Async(true),
	}
	s.collector = colly.NewCollector(collectorOptions...)
	s.collector.SetRequestTimeout(30 * time.Second)

	err := s.collector.Limit(&colly.LimitRule{
		Parallelism: int(s.config.Parallelism),
		RandomDelay: s.config.Delay,
		DomainGlob:  "*",
	})
	if err != nil {
		return err
	}

	if s.config.NeedsHeadlessBrowser {
		mango.Log("Using headless browser")
		transport := headless.GetTransport(s.options.Headless, s.config.LocalStorage, s.config.GetActions())
		s.collector.WithTransport(transport)
	}
	return nil
}
