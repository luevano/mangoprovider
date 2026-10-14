package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

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

func TestCloudflareFallback(t *testing.T) {
	fs, cmds := fakeFlaresolverr(t, "<html>solved</html>")
	s := site(t)
	c, err := NewClient(nil, mango.HeadlessOptions{UseFlaresolverr: true, FlaresolverrURL: fs.URL}, nil, Config{Strategy: CloudflareFallback})
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

func TestCloudflareWithoutFlaresolverr(t *testing.T) {
	s := site(t)
	c, err := NewClient(nil, mango.HeadlessOptions{}, nil, Config{Strategy: CloudflareFallback})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = get(t, c, s.URL+"/challenged")
	if !errors.Is(err, ErrChallenged) || !errors.Is(err, mango.ErrSiteDown) || !strings.Contains(err.Error(), "use_flaresolverr") {
		t.Errorf("got %v, want ErrChallenged and ErrSiteDown", err)
	}
	if err := c.Close(); err != nil {
		t.Errorf("close without a session: %v", err)
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
	c, err := NewClient(nil, mango.HeadlessOptions{UseFlaresolverr: true, FlaresolverrURL: fs.URL}, nil, Config{Strategy: CloudflareFallback})
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
