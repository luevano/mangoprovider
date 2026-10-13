package scraper

import (
	"context"
	"net/http"
	"time"

	mango "github.com/luevano/mangoprovider"
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
	client  *http.Client
	log     *mango.Logger
	limiter *rate.Limiter
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
	client := options.HTTPClient
	if client == nil {
		client = mango.DefaultOptions().HTTPClient
	}
	// Set the new BaseURL if there are redirects.
	err = setBaseURLOnRedirect(ctx, config, client)
	if err != nil {
		return nil, err
	}

	return &Scraper{
		config:  config,
		options: options,
		client:  client,
		log:     log,
		limiter: rate.NewLimiter(rate.Every(requestInterval), requestBurst),
		slots:   make(chan struct{}, config.Parallelism),
	}, nil
}
