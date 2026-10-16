package scraper

import (
	"context"
	"time"

	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/fetch"
	"github.com/luevano/mangoprovider/scraper/headless/rod"
	"golang.org/x/time/rate"
)

// Every scrape is one html request (images go through GetPageImage, not the scraper), these
// keep many of them at once (a library update, a batch of chapters) polite: at most 4 at a time,
// ~4 a second after a burst of 4. A single search or chapter list isn't slowed down.
const (
	maxParallelism  = 4
	requestInterval = 250 * time.Millisecond
	requestBurst    = 4
)

// Scraper: Generic scraper downloads html pages and parses them.
type Scraper struct {
	config  *Configuration
	options mango.Options
	client  *fetch.Client
	log     *mango.Logger
	// slots: one per request in flight, Parallelism of them.
	slots chan struct{}
}

// NewScraper: generates a new scraper with given configuration and options.
func NewScraper(ctx context.Context, config *Configuration, options mango.Options, log *mango.Logger) (scraper *Scraper, err error) {
	// Set the parallelism if not set by the scraper, use the provided parallelism option
	if config.Parallelism == 0 {
		config.Parallelism = max(min(options.Parallelism, maxParallelism), 1)
		log.Log("setting parallelism to %d", config.Parallelism)
	}
	limit, burst := rate.Every(requestInterval), requestBurst
	if config.RateLimit != 0 {
		limit, burst = config.RateLimit, max(config.RateBurst, 1)
	}
	retry := fetch.DefaultRetry()
	if config.Retry != nil {
		retry = *config.Retry
	}
	client, err := fetch.NewClient(options.HTTPClient, options.Headless, log, fetch.Config{
		Strategy:    config.Fetch,
		Headless:    rod.Options{LoadWait: config.LoadWait, LocalStorage: config.LocalStorage},
		Limiter:     rate.NewLimiter(limit, burst),
		Middlewares: config.Middlewares,
		Retry:       &retry,
	})
	if err != nil {
		return nil, err
	}
	return &Scraper{
		config:  config,
		options: options,
		client:  client,
		log:     log,
		slots:   make(chan struct{}, config.Parallelism),
	}, nil
}

// Close: Ends what the fetch strategy started (FlareSolverr session, browser).
func (s *Scraper) Close() error {
	return s.client.Close()
}
