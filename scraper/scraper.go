package scraper

import (
	"context"
	"time"

	"github.com/gocolly/colly/v2"
	mango "github.com/luevano/mangoprovider"
)

// Scraper: Generic scraper downloads html pages and parses them.
type Scraper struct {
	config    *Configuration
	options   mango.Options
	collector *colly.Collector
	log       *mango.Logger
}

// NewScraper: generates a new scraper with given configuration and options.
func NewScraper(ctx context.Context, config *Configuration, options mango.Options, log *mango.Logger) (scraper *Scraper, err error) {
	// Set the parallelism if not set by the scraper, use the provided parallelism option
	if config.Parallelism == 0 {
		log.Log("setting parallelism to %d", options.Parallelism)
		config.Parallelism = options.Parallelism
	}
	// Set the new BaseURL if there are redirects.
	err = setBaseURLOnRedirect(ctx, config, options.HTTPClient)
	if err != nil {
		return nil, err
	}

	scraper = &Scraper{
		config:  config,
		options: options,
		log:     log,
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
	s.collector.SetRequestTimeout(60 * time.Second)

	return s.collector.Limit(&colly.LimitRule{
		Parallelism: int(s.config.Parallelism),
		RandomDelay: s.config.Delay,
		DomainGlob:  "*",
	})
}
