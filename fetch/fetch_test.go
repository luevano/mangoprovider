package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/launcher"
	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/logger"
	"github.com/luevano/libmangal/progress"
	mango "github.com/luevano/mangoprovider"
	"golang.org/x/time/rate"
)

const challenge = `<!DOCTYPE html><html><head><title>Just a moment...</title></head></html>`

// fakeFlaresolverr answers every request.get with body for the url, and records the commands.
func fakeFlaresolverr(t *testing.T, body string) (*httptest.Server, *[]string) {
	t.Helper()
	var mu sync.Mutex
	var cmds []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Cmd, URL string }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		mu.Lock()
		cmds = append(cmds, req.Cmd+" "+req.URL)
		mu.Unlock()
		out, _ := json.Marshal(map[string]any{"status": "ok", "solution": map[string]any{
			"url": req.URL, "status": 200, "headers": map[string]string{}, "response": body,
		}})
		w.Write(out)
	}))
	t.Cleanup(server.Close)
	return server, &cmds
}

// site serves a challenge on /challenged, a plain 403 on /forbidden, "ok" elsewhere.
func site(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/challenged":
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(challenge))
		case "/mitigated":
			w.Header().Set("Cf-Mitigated", "challenge")
			w.WriteHeader(http.StatusServiceUnavailable)
		case "/forbidden":
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("no"))
		default:
			w.Write([]byte("ok"))
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func get(t *testing.T, c *Client, url string) (int, string, error) {
	t.Helper()
	resp, err := c.Get(url)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), err
}

func TestHeadlessFallback(t *testing.T) {
	fs, cmds := fakeFlaresolverr(t, "<html>solved</html>")
	s := site(t)
	c, err := NewClient(nil, mango.HeadlessOptions{UseFlaresolverr: true, FlaresolverrURL: fs.URL}, nil, Config{
		Strategy:   HeadlessFallback,
		NewBrowser: func() Browser { t.Error("browser started with flaresolverr on"); return nil },
	})
	if err != nil {
		t.Fatal(err)
	}

	for path, want := range map[string]string{"/challenged": "<html>solved</html>", "/mitigated": "<html>solved</html>", "/open": "ok"} {
		status, body, err := get(t, c, s.URL+path)
		if err != nil || status != 200 || body != want {
			t.Errorf("%s: got %d %q, %v, want %q", path, status, body, err, want)
		}
	}
	// a 403 that isn't a challenge stays a 403, body intact
	if status, body, err := get(t, c, s.URL+"/forbidden"); err != nil || status != 403 || body != "no" {
		t.Errorf("/forbidden: got %d %q, %v", status, body, err)
	}

	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	want := []string{"request.get " + s.URL + "/challenged", "request.get " + s.URL + "/mitigated", "sessions.destroy "}
	if strings.Join(*cmds, ",") != strings.Join(want, ",") && strings.Join(*cmds, ",") != strings.Join([]string{want[1], want[0], want[2]}, ",") {
		t.Errorf("flaresolverr got %q, want %q", *cmds, want)
	}
}

// fakeBrowser answers every request with f, counts its requests and records Close.
type fakeBrowser struct {
	f        RoundTripperFunc
	requests int
	closed   bool
}

func (b *fakeBrowser) RoundTrip(r *http.Request) (*http.Response, error) {
	b.requests++
	return b.f(r)
}

func (b *fakeBrowser) Close() error {
	b.closed = true
	return nil
}

func page(r *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)), Request: r}
}

func TestHeadlessFallbackBrowser(t *testing.T) {
	s := site(t)
	var started int
	browser := &fakeBrowser{f: func(r *http.Request) (*http.Response, error) { return page(r, 200, "<html>rendered</html>"), nil }}
	c, err := NewClient(nil, mango.HeadlessOptions{}, nil, Config{
		Strategy:   HeadlessFallback,
		NewBrowser: func() Browser { started++; return browser },
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, body, err := get(t, c, s.URL+"/open"); err != nil || body != "ok" || started != 0 {
		t.Fatalf("got %q, %v, browser started %d times, want direct", body, err, started)
	}
	for range 2 {
		if status, body, err := get(t, c, s.URL+"/challenged"); err != nil || status != 200 || body != "<html>rendered</html>" {
			t.Errorf("got %d %q, %v", status, body, err)
		}
	}
	if started != 1 || browser.requests != 2 {
		t.Errorf("browser started %d times for %d requests, want once for 2", started, browser.requests)
	}
	if err := c.Close(); err != nil || !browser.closed {
		t.Errorf("close: %v, browser closed %t", err, browser.closed)
	}
}

func TestHeadlessFallbackUnavailable(t *testing.T) {
	s := site(t)
	for name, f := range map[string]RoundTripperFunc{
		"no browser": func(*http.Request) (*http.Response, error) { return nil, errors.New("launch browser: no chromium") },
		"challenged": func(r *http.Request) (*http.Response, error) { return page(r, 403, challenge), nil },
	} {
		c, err := NewClient(nil, mango.HeadlessOptions{}, nil, Config{
			Strategy:   HeadlessFallback,
			NewBrowser: func() Browser { return &fakeBrowser{f: f} },
		})
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = get(t, c, s.URL+"/challenged")
		if !errors.Is(err, ErrChallenged) || !errors.Is(err, mango.ErrSiteDown) || !strings.Contains(err.Error(), "use_flaresolverr") {
			t.Errorf("%s: got %v, want ErrChallenged and ErrSiteDown", name, err)
		}
		if err := c.Close(); err != nil {
			t.Errorf("%s: close: %v", name, err)
		}
	}
}

func TestHeadlessFallbackWarnsOnce(t *testing.T) {
	fs, _ := fakeFlaresolverr(t, "<html>solved</html>")
	s := site(t)
	other := site(t)
	log := mango.NewProviderLogger(libmangal.ProviderInfo{ID: "mango-test", Name: "Test"})
	var lines []string
	l := logger.NewLogger()
	l.SetOnLog(func(format string, a ...any) { lines = append(lines, fmt.Sprintf(format, a...)) })
	log.SetLogger(l)
	var reports []string
	ctx := progress.WithReporter(context.Background(), func(s progress.Status) {
		if !s.Minor {
			reports = append(reports, s.Message)
		}
	})
	c, err := NewClient(nil, mango.HeadlessOptions{UseFlaresolverr: true, FlaresolverrURL: fs.URL}, log, Config{Strategy: HeadlessFallback})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	for _, u := range []string{s.URL + "/challenged", s.URL + "/mitigated", other.URL + "/challenged"} {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		resp, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}

	warning := func(s string) bool { return strings.Contains(s, "fetched through FlareSolverr") }
	var warnings []string
	for _, r := range reports {
		if warning(r) {
			warnings = append(warnings, r)
		}
	}
	want := "Test: challenged at " + mango.Host(s.URL) + ", fetched through FlareSolverr, consider fetch.Headless for this site"
	if len(warnings) != 2 || warnings[0] != want || !strings.Contains(warnings[1], mango.Host(other.URL)) {
		t.Errorf("got warnings %q, want once per host, the first %q", warnings, want)
	}
	var logged int
	for _, line := range lines {
		if warning(line) {
			logged++
		}
	}
	if logged != 3 {
		t.Errorf("logged %d fallbacks, want every one (3): %q", logged, lines)
	}
}

func TestHeadlessFallbackRod(t *testing.T) {
	if testing.Short() || launcher.NewBrowser().Validate() != nil {
		t.Skip("needs rod's chromium already downloaded")
	}
	// challenges what doesn't ask for html like a browser does
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept"), "text/html") {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(challenge))
			return
		}
		w.Write([]byte(`<html><body><p id="r"></p><script>document.getElementById("r").textContent = "rendered"</script></body></html>`))
	}))
	defer s.Close()
	c, err := NewClient(nil, mango.HeadlessOptions{}, nil, Config{Strategy: HeadlessFallback})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if status, body, err := get(t, c, s.URL); err != nil || status != 200 || !strings.Contains(body, `<p id="r">rendered</p>`) {
		t.Errorf("got %d %q, %v", status, body, err)
	}
}

func TestDirectKeepsChallenge(t *testing.T) {
	s := site(t)
	c, err := NewClient(nil, mango.HeadlessOptions{UseFlaresolverr: true, FlaresolverrURL: "http://flaresolverr.invalid"}, nil, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if status, body, err := get(t, c, s.URL+"/challenged"); err != nil || status != 403 || body != challenge {
		t.Errorf("got %d %q, %v", status, body, err)
	}
}

func TestFlaresolverrFails(t *testing.T) {
	s := site(t)
	fs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status": "error", "message": "Error solving the challenge. Timeout after 60.0 seconds."}`))
	}))
	defer fs.Close()
	c, err := NewClient(nil, mango.HeadlessOptions{UseFlaresolverr: true, FlaresolverrURL: fs.URL}, nil, Config{Strategy: HeadlessFallback})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := get(t, c, s.URL+"/challenged"); !errors.Is(err, mango.ErrSiteDown) {
		t.Errorf("got %v, want ErrSiteDown", err)
	}
}

func TestMiddlewares(t *testing.T) {
	var got http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
	}))
	defer server.Close()

	var order []string
	mark := func(name string) Middleware {
		return func(next http.RoundTripper) http.RoundTripper {
			return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
				order = append(order, name)
				return next.RoundTrip(r)
			})
		}
	}
	c, err := NewClient(nil, mango.HeadlessOptions{}, nil, Config{Middlewares: []Middleware{
		mark("first"),
		Headers(map[string]string{"Referer": "https://site/", "X-Site": "1"}),
		mark("second"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	req.Header.Set("Referer", "https://site/manga")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got.Get("Referer") != "https://site/manga" || got.Get("X-Site") != "1" {
		t.Errorf("got headers %v, want the request's Referer kept and X-Site added", got)
	}
	if req.Header.Get("X-Site") != "" {
		t.Error("the caller's request was changed")
	}
	if strings.Join(order, ",") != "first,second" {
		t.Errorf("got order %v", order)
	}
}

func TestRateLimit(t *testing.T) {
	s := site(t)
	c, err := NewClient(nil, mango.HeadlessOptions{}, nil, Config{Limiter: rate.NewLimiter(rate.Every(100*time.Millisecond), 1)})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	for range 3 {
		if _, _, err := get(t, c, s.URL); err != nil {
			t.Fatal(err)
		}
	}
	if elapsed := time.Since(start); elapsed < 200*time.Millisecond {
		t.Errorf("3 requests took %s, want at least 200ms", elapsed)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, s.URL, nil)
	if _, err := c.Do(req); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
}

func TestOwnJar(t *testing.T) {
	base := &http.Client{}
	c, err := NewClient(base, mango.HeadlessOptions{}, nil, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if base.Jar != nil || c.Jar == nil || c.Direct().Jar != c.Jar {
		t.Error("wanted a jar on the copy only, shared with Direct")
	}
}

func TestHeadlessDebug(t *testing.T) {
	var lines []string
	browser := &fakeBrowser{f: func(r *http.Request) (*http.Response, error) { return page(r, 200, "<html></html>"), nil }}
	c, err := NewClient(nil, mango.HeadlessOptions{}, nil, Config{
		Strategy:   Headless,
		NewBrowser: func() Browser { return browser },
		Debug:      func(format string, a ...any) { lines = append(lines, fmt.Sprintf(format, a...)) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := get(t, c, "http://example.com/page?token=x"); err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "headless GET http://example.com/page?token=REDACTED 200 ") {
		t.Errorf("got %q", lines)
	}
	if err := c.Close(); err != nil || !browser.closed {
		t.Errorf("close: %v, browser closed %t", err, browser.closed)
	}
}
