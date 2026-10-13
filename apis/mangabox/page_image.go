package mangabox

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"

	mango "github.com/luevano/mangoprovider"
)

// GetPageImage tries the image on each cdn the chapter lists, starting with the one that
// served the last image (like keiyoushi's useAltCdnInterceptor).
func (m *mangabox) GetPageImage(ctx context.Context, client *http.Client, page mango.Page) ([]byte, error) {
	var errs []error
	urls := m.cdnOrder(append([]string{page.URL}, page.Fallbacks...))
	for i, u := range urls {
		page.URL = u
		image, err := mango.GenericGetPageImage(ctx, client, page)
		if err == nil {
			m.setCDN(origin(u))
			return image, nil
		}
		if ctx.Err() != nil {
			return nil, err
		}
		if i+1 < len(urls) && m.cdnFailed(origin(u)) {
			m.log.Status(ctx, "image cdn %s failed (%s), trying %s", mango.Host(u), mango.Reason(err), mango.Host(urls[i+1]))
		}
		errs = append(errs, err)
	}
	return nil, errors.Join(errs...)
}

// cdnOrder moves the url on the cdn that worked last to the front.
func (m *mangabox) cdnOrder(urls []string) []string {
	m.mu.Lock()
	cdn := m.cdn
	m.mu.Unlock()
	if i := slices.IndexFunc(urls, func(u string) bool { return origin(u) == cdn }); i > 0 {
		first := urls[i]
		copy(urls[1:i+1], urls[:i])
		urls[0] = first
	}
	return urls
}

func (m *mangabox) setCDN(cdn string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cdn = cdn
	delete(m.cdnDown, cdn)
}

// cdnFailed marks the cdn as failing, true the first time, so a switch is reported once and
// not for every page.
func (m *mangabox) cdnFailed(cdn string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cdnDown[cdn] {
		return false
	}
	m.cdnDown[cdn] = true
	return true
}

func origin(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
