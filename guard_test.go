package mangoprovider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"code.dny.dev/ssrf"
	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/netutil"
)

func TestGuardClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("png"))
	}))
	defer server.Close()
	get := func(client *http.Client) error {
		resp, err := client.Get(server.URL)
		if err == nil {
			resp.Body.Close()
		}
		return err
	}

	o := DefaultOptions()
	if err := get(GuardClient(o.HTTPClient, o)); !errors.Is(err, ssrf.ErrProhibitedIP) {
		t.Errorf("loopback: got %v, want refused", err)
	}
	if o.HTTPClient.Transport != nil {
		t.Error("the given client was changed")
	}

	o.Mirrors = map[string]string{"mango-test": server.URL + "/"}
	if err := get(GuardClient(o.HTTPClient, o)); err != nil {
		t.Errorf("a configured mirror: %v", err)
	}

	o = DefaultOptions()
	o.AllowPrivateAddresses = true
	if err := get(GuardClient(o.HTTPClient, o)); err != nil {
		t.Errorf("allowed: %v", err)
	}

	// a logging transport is guarded underneath, and still logs
	logged := 0
	o = DefaultOptions()
	o.HTTPClient = &http.Client{Transport: netutil.NewLoggingTransport(nil, func(string, ...any) { logged++ }, nil)}
	if err := get(GuardClient(o.HTTPClient, o)); !errors.Is(err, ssrf.ErrProhibitedIP) {
		t.Errorf("logging transport: got %v, want refused", err)
	}
	if logged != 1 {
		t.Errorf("logged %d requests, want 1", logged)
	}
	if o.HTTPClient.Transport.(*netutil.LoggingTransport).Next != nil {
		t.Error("the given logging transport was changed")
	}

	// a transport it can't guard stays
	custom := &http.Client{Transport: RoundTripperFunc(nil)}
	if GuardClient(custom, DefaultOptions()) != custom {
		t.Error("a custom transport should be kept")
	}
}

// RoundTripperFunc for the test, mangoprovider has none.
type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (f RoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestLoadGuardsPageImages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("png"))
	}))
	defer server.Close()

	load := func(o Options) libmangal.Provider {
		l := &Loader{
			ProviderInfo: libmangal.ProviderInfo{ID: "mango-test", Name: "Test", Version: "0.1.0"},
			Options:      o,
			F:            func(context.Context, *Logger) (Functions, error) { return Functions{}, nil },
		}
		p, err := l.Load(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	page := &Page{URL: server.URL + "/1.png", Chapter_: &Chapter{Title: "Ch. 1"}}

	if _, err := load(DefaultOptions()).GetPageImage(context.Background(), page); !errors.Is(err, ssrf.ErrProhibitedIP) {
		t.Errorf("a site's image on loopback: got %v, want refused", err)
	}
	o := DefaultOptions()
	o.AllowPrivateAddresses = true
	if image, err := load(o).GetPageImage(context.Background(), page); err != nil || string(image) != "png" {
		t.Errorf("allowed: %q, %v", image, err)
	}
}
