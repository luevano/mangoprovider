package scraper

import (
	"context"
	"time"

	"github.com/gocolly/colly/v2"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/scraper/headless"
)

// Scraper: Generic scraper downloads html pages and parses them.
type Scraper struct {
	config    *Configuration
	options   mango.Options
	collector *colly.Collector
	log       *mango.Logger
	// only set for configurations that need a headless browser
	transport headless.Transport
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
	err = scraper.setCollector(ctx)
	if err != nil {
		return nil, err
	}

	return scraper, nil
}

func (s *Scraper) setCollector(ctx context.Context) error {
	collectorOptions := []colly.CollectorOption{
		colly.AllowURLRevisit(),
		colly.Async(true),
	}
	s.collector = colly.NewCollector(collectorOptions...)
	s.collector.SetRequestTimeout(60 * time.Second)

	err := s.collector.Limit(&colly.LimitRule{
		Parallelism: int(s.config.Parallelism),
		RandomDelay: s.config.Delay,
		DomainGlob:  "*",
	})
	if err != nil {
		return err
	}

	if s.config.NeedsHeadlessBrowser {
		s.log.Log("using headless browser")
		s.transport = headless.NewTransport(
			ctx,
			s.options.Headless,
			s.config.LoadWait,
			s.config.LocalStorage,
			s.config.GetActions(),
			s.log)
		s.collector.WithTransport(s.transport)
	}
	return nil
}

// Close: Closes the headless browser, if any.
func (s *Scraper) Close() error {
	if s.transport == nil {
		return nil
	}
	return s.transport.Close()
}
