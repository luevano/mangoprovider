package rod

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
)

func needsChromium(t *testing.T) {
	t.Helper()
	if testing.Short() || launcher.NewBrowser().Validate() != nil {
		t.Skip("needs rod's chromium already downloaded")
	}
}

func newSite(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/old":
			http.Redirect(w, r, "/new", http.StatusFound)
			return
		case "/missing":
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`<html><body>not here</body></html>`))
			return
		case "/limited":
			w.Header().Set("Retry-After", "7")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		var cookies []string
		for _, c := range r.Cookies() {
			cookies = append(cookies, c.Name+"="+c.Value)
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><p id="path">` + r.URL.Path + `</p><p id="cookies">` + strings.Join(cookies, ",") +
			`</p><p id="referer">` + r.Referer() + `</p><p id="js"></p><script>document.getElementById("js").textContent = "ran"</script></body></html>`))
	}))
	t.Cleanup(server.Close)
	return server
}

func roundTrip(t *testing.T, tr *TransportRod, req *http.Request) (*http.Response, string) {
	t.Helper()
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, string(body)
}

func TestRoundTrip(t *testing.T) {
	needsChromium(t)
	site := newSite(t)
	tr := NewTransport(Options{}, nil)
	defer tr.Close()

	req, _ := http.NewRequest(http.MethodGet, site.URL+"/page", nil)
	req.Header.Set("Cookie", "a=1; b=2")
	req.Header.Set("Referer", site.URL+"/")
	resp, body := roundTrip(t, tr, req)
	for _, want := range []string{`<p id="cookies">a=1,b=2</p>`, `<p id="referer">` + site.URL + `/</p>`, `<p id="js">ran</p>`} {
		if !strings.Contains(body, want) {
			t.Errorf("body %s: missing %s", body, want)
		}
	}
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
		t.Errorf("got %d, %v", resp.StatusCode, resp.Header)
	}
}

func TestRoundTripStatus(t *testing.T) {
	needsChromium(t)
	site := newSite(t)
	tr := NewTransport(Options{}, nil)
	defer tr.Close()

	req, _ := http.NewRequest(http.MethodGet, site.URL+"/missing", nil)
	if resp, _ := roundTrip(t, tr, req); resp.StatusCode != http.StatusNotFound {
		t.Errorf("got %d, want the page's 404", resp.StatusCode)
	}
	req, _ = http.NewRequest(http.MethodGet, site.URL+"/limited", nil)
	if resp, _ := roundTrip(t, tr, req); resp.StatusCode != http.StatusTooManyRequests || resp.Header.Get("Retry-After") != "7" {
		t.Errorf("got %d, %v, want the 429 with its Retry-After", resp.StatusCode, resp.Header)
	}
	// the final url after redirects, like a direct request
	req, _ = http.NewRequest(http.MethodGet, site.URL+"/old", nil)
	resp, body := roundTrip(t, tr, req)
	if resp.StatusCode != http.StatusOK || resp.Request == nil || resp.Request.URL.String() != site.URL+"/new" || !strings.Contains(body, "/new") {
		t.Errorf("got %d, %v, want 200 from %s/new", resp.StatusCode, resp.Request, site.URL)
	}
}

func TestAction(t *testing.T) {
	needsChromium(t)
	site := newSite(t)
	tr := NewTransport(Options{}, nil)
	defer tr.Close()

	ctx := WithAction(context.Background(), func(p *rod.Page) error {
		_, err := p.Eval(`() => document.getElementById("path").textContent = "changed"`)
		return err
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, site.URL+"/page", nil)
	if _, body := roundTrip(t, tr, req); !strings.Contains(body, `<p id="path">changed</p>`) {
		t.Errorf("the action didn't run: %s", body)
	}
}

// a failed launch isn't kept, the next request tries again
func TestLaunchRetried(t *testing.T) {
	needsChromium(t)
	site := newSite(t)
	tr := NewTransport(Options{}, nil)
	defer tr.Close()

	tr.newLauncher = func() *launcher.Launcher { return launcher.New().Bin("/nonexistent/chromium") }
	req, _ := http.NewRequest(http.MethodGet, site.URL+"/page", nil)
	if _, err := tr.RoundTrip(req); err == nil {
		t.Fatal("wanted the launch to fail")
	}
	tr.newLauncher = launcher.New
	roundTrip(t, tr, req)
}

// a cancelled request doesn't launch the browser
func TestLaunchCanceled(t *testing.T) {
	needsChromium(t)
	tr := NewTransport(Options{}, nil)
	defer tr.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1/", nil)
	if _, err := tr.RoundTrip(req); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
	if tr.browser != nil {
		t.Error("a cancelled request launched the browser")
	}
}

// Close stops the browser and removes its user data dir
func TestClose(t *testing.T) {
	needsChromium(t)
	site := newSite(t)
	tr := NewTransport(Options{}, nil)
	req, _ := http.NewRequest(http.MethodGet, site.URL+"/page", nil)
	roundTrip(t, tr, req)

	dir := tr.launcher.Get(flags.UserDataDir)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("user data dir: %v", err)
	}
	if err := tr.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("user data dir %s left behind: %v", dir, err)
	}
	if _, err := tr.RoundTrip(req); err == nil {
		t.Error("a closed transport launched again")
	}
	if err := tr.Close(); err != nil {
		t.Errorf("second close: %v", err)
	}
}

// Close during the launch stops it, no browser left behind
func TestCloseWhileLaunching(t *testing.T) {
	needsChromium(t)
	site := newSite(t)
	tr := NewTransport(Options{}, nil)
	launched := make(chan *launcher.Launcher, 1)
	tr.newLauncher = func() *launcher.Launcher {
		l := launcher.New()
		launched <- l
		return l
	}

	done := make(chan error, 1)
	go func() {
		req, _ := http.NewRequest(http.MethodGet, site.URL+"/page", nil)
		_, err := tr.RoundTrip(req)
		done <- err
	}()
	l := <-launched
	if err := tr.Close(); err != nil {
		t.Error(err)
	}
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("the request didn't end")
	}
	if dir := l.Get(flags.UserDataDir); dir != "" {
		time.Sleep(100 * time.Millisecond)
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("user data dir %s left behind: %v", dir, err)
		}
	}
}

func TestRequestCookies(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://site/manga", nil)
	req.Header.Set("Cookie", "a=1; b=x=y")
	var got []string
	for _, c := range requestCookies(req) {
		got = append(got, c.Name+"="+c.Value+" "+c.URL)
	}
	if want := "a=1 https://site/manga,b=x=y https://site/manga"; strings.Join(got, ",") != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
