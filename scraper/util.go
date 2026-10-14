package scraper

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	mango "github.com/luevano/mangoprovider"
	whatwgurl "github.com/nlnwa/whatwg-url/url"
	"golang.org/x/net/html/charset"
)

// Checks redirections and sets the new BaseURL if needed.
func setBaseURLOnRedirect(ctx context.Context, config *Configuration, httpClient *http.Client) error {
	// same client (timeout, transport), just without following redirects
	client := *httpClient
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, config.BaseURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("check %s for redirects: %w", config.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusFound {
		loc, err := resp.Location()
		if err != nil {
			return err
		}
		config.BaseURL = loc.String()
	}
	return nil
}

// urlParser: Resolves the urls in the pages like browsers do (WHATWG URL standard, as colly
// did): Go's url.Parse refuses some that sites use (a lone "%", tabs in the path). A lone "%"
// becomes "%25" so Go can request the result.
var urlParser = whatwgurl.NewParser(whatwgurl.WithPercentEncodeSinglePercentSign())

// page: A scraped html page.
type page struct {
	doc *goquery.Document
	// url: Where the page ended up, after the redirects.
	url *url.URL
	// base: What its relative urls are relative to (url or its <base href>).
	base string
}

// absoluteURL: u resolved against the page, empty for fragments ("#x") and urls that don't parse.
func (p *page) absoluteURL(u string) string {
	u = strings.TrimSpace(u)
	if strings.HasPrefix(u, "#") {
		return ""
	}
	abs, err := urlParser.ParseRef(p.base, u)
	if err != nil {
		return ""
	}
	return abs.Href(false)
}

// scrape: GETs rawURL, see scrapeRequest.
func (s *Scraper) scrape(ctx context.Context, rawURL, referer string, parse func(*page) error) error {
	return s.scrapeRequest(ctx, Request{URL: rawURL}, referer, parse)
}

// scrapeRequest: Sends r (throttled, at most Parallelism at once) with referer and parses the
// html with parse. Any status but 200 is an error (see mango.StatusError), so is a panic in parse
// (e.g. a site config indexing a url that changed), instead of crashing the whole program.
func (s *Scraper) scrapeRequest(ctx context.Context, r Request, referer string, parse func(*page) error) (err error) {
	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := s.delay(ctx); err != nil {
		return err
	}

	rawURL, method := r.URL, http.MethodGet
	var form io.Reader
	if r.Form != nil {
		method, form = http.MethodPost, strings.NewReader(r.Form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, form)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, rawURL, err)
	}
	if r.Form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.Header.Set("Referer", referer)
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("Accept", "text/html")
	req.Header.Set("User-Agent", s.options.UserAgent)
	// custom scraper headers (like "Cookie" for some sites)
	for k, v := range s.config.Headers {
		req.Header.Set(k, v)
	}
	for k, v := range r.Headers {
		req.Header.Set(k, v)
	}

	body, resp, err := mango.FetchRequest(s.client.Client, nil, req)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, rawURL, err)
	}
	if err := mango.StatusError(resp.StatusCode, resp.Header); err != nil {
		return fmt.Errorf("%s %s: %w", method, rawURL, err)
	}

	p := &page{url: resp.Request.URL}
	if p.doc, err = parseHTML(body, resp.Header.Get("Content-Type")); err != nil {
		return fmt.Errorf("parse %s: %w", p.url, err)
	}
	p.base = p.url.String()
	if href, ok := p.doc.Find("base[href]").First().Attr("href"); ok {
		if base, err := urlParser.ParseRef(p.base, href); err == nil {
			p.base = base.Href(false)
		}
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("parse %s: %v", p.url, r)
		}
	}()
	return parse(p)
}

// delay: Waits a random time up to the configured Delay.
func (s *Scraper) delay(ctx context.Context) error {
	if s.config.Delay <= 0 {
		return nil
	}
	timer := time.NewTimer(rand.N(s.config.Delay))
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// parseHTML: The html in body, decoded to UTF-8 if the page declares another charset (in the
// Content-Type, a BOM or a <meta>). An undeclared one is UTF-8.
func parseHTML(body []byte, contentType string) (*goquery.Document, error) {
	if enc, name, certain := charset.DetermineEncoding(body, contentType); certain {
		if name == "utf-8" {
			return goquery.NewDocumentFromReader(bytes.NewReader(body))
		}
		return goquery.NewDocumentFromReader(enc.NewDecoder().Reader(bytes.NewReader(body)))
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	label := metaCharset(doc)
	if label == "" {
		return doc, nil
	}
	enc, name := charset.Lookup(label)
	if enc == nil || name == "utf-8" {
		return doc, nil
	}
	return goquery.NewDocumentFromReader(enc.NewDecoder().Reader(bytes.NewReader(body)))
}

// metaCharset: The charset declared by a <meta charset> or <meta http-equiv="Content-Type">.
func metaCharset(doc *goquery.Document) string {
	if cs, ok := doc.Find("meta[charset]").First().Attr("charset"); ok {
		return strings.TrimSpace(cs)
	}
	var label string
	doc.Find("meta[http-equiv]").EachWithBreak(func(_ int, meta *goquery.Selection) bool {
		if !strings.EqualFold(meta.AttrOr("http-equiv", ""), "content-type") {
			return true
		}
		_, cs, ok := strings.Cut(strings.ToLower(meta.AttrOr("content", "")), "charset=")
		if ok {
			label = strings.Trim(strings.TrimSpace(cs), `"'`)
		}
		return !ok
	})
	return label
}
