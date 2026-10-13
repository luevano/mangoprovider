package scraper

import (
	"context"
	"time"

	"github.com/gocolly/colly/v2"
	mango "github.com/luevano/mangoprovider"
	"golang.org/x/time/rate"
)

// Every scrape is one html request (images go through GetPageImage, not the collector), these
// keep many of them at once (a library update, a batch of chapters) polite: at most 4 at a time,
// ~4 a second after a burst of 4. A single search or chapter list isn't slowed down.
const (
	maxParallelism  = 4
	requestInterval = 250 * time.Millisecond
	requestBurst    = 4
)

// Scraper: Generic scraper downloads html pages and parses them.
type Scraper struct {
	config    *Configuration
	options   mango.Options
	collector *colly.Collector
	log       *mango.Logger
	limiter   *rate.Limiter
}

// NewScraper: generates a new scraper with given configuration and options.
func NewScraper(ctx context.Context, config *Configuration, options mango.Options, log *mango.Logger) (scraper *Scraper, err error) {
	// Set the parallelism if not set by the scraper, use the provided parallelism option
	if config.Parallelism == 0 {
		config.Parallelism = min(options.Parallelism, maxParallelism)
		log.Log("setting parallelism to %d", config.Parallelism)
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
		limiter: rate.NewLimiter(rate.Every(requestInterval), requestBurst),
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
