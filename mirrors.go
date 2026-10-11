package mangoprovider

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/luevano/libmangal"
)

// Mirrors picks the website a provider uses among its Website and Mirrors.
//
// Order: the one that worked earlier in this run, the one that last worked (cached, until it
// expires), the preferred one (Options.Mirrors), Website, then the other Mirrors. Only
// ErrSiteDown moves on to the next.
type Mirrors struct {
	id        string
	sites     []string
	preferred string
	log       *Logger

	mu      sync.Mutex
	current string
}

func NewMirrors(info libmangal.ProviderInfo, preferred string, log *Logger) *Mirrors {
	sites := []string{info.Website}
	for _, mirror := range info.Mirrors {
		if !slices.Contains(sites, mirror) {
			sites = append(sites, mirror)
		}
	}
	return &Mirrors{id: info.ID, sites: sites, preferred: preferred, log: log}
}

// Default is the preferred website, else Website.
func (m *Mirrors) Default() string {
	if m.preferred != "" {
		return m.preferred
	}
	return m.sites[0]
}

// Do calls f with each website until one isn't down; the one that works is remembered, and
// cached unless it's the default (a cached one expires, then the default is tried first again).
func (m *Mirrors) Do(ctx context.Context, store Store, f func(website string) error) error {
	cached, hasCached, err := store.GetMirror(m.id)
	if err != nil {
		m.log.Log("mirrors: %s", err)
	}
	m.mu.Lock()
	current := m.current
	m.mu.Unlock()

	var order []string
	for _, site := range append([]string{current, cached, m.Default()}, m.sites...) {
		if site != "" && !slices.Contains(order, site) {
			order = append(order, site)
		}
	}

	var errs []error
	for _, site := range order {
		err := f(site)
		if err == nil {
			m.mu.Lock()
			m.current = site
			m.mu.Unlock()
			if site != m.Default() {
				err = store.SetMirror(m.id, site)
			} else if hasCached {
				err = store.DeleteMirror(m.id)
			}
			if err != nil {
				m.log.Log("mirrors: %s", err)
			}
			return nil
		}
		if ctx.Err() != nil || !errors.Is(err, ErrSiteDown) || len(order) == 1 {
			return err
		}
		m.log.Log("mirrors: %s is down, trying the next one: %s", site, err)
		errs = append(errs, fmt.Errorf("%s: %w", site, err))
	}
	return fmt.Errorf("every mirror is down: %w", errors.Join(errs...))
}
