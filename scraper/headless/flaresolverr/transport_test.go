package flaresolverr

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRoundTripTimeout(t *testing.T) {
	defer func(m, r time.Duration) { defaultMaxTimeout, requestMargin = m, r }(defaultMaxTimeout, requestMargin)
	defaultMaxTimeout, requestMargin = 100*time.Millisecond, 100*time.Millisecond

	hang := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-hang
	}))
	defer server.Close()
	defer close(hang)

	transport := NewTransport(server.URL)
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)

	done := make(chan error, 1)
	go func() {
		_, err := transport.RoundTrip(req)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Error("wanted a timeout error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("a flaresolverr that never answers should time out")
	}
}

func TestRoundTripUsesClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok","solution":{"status":200,"response":"<html></html>"}}`))
	}))
	defer server.Close()

	used := 0
	client := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		used++
		return http.DefaultTransport.RoundTrip(r)
	})}
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	resp, err := NewTransportClient(server.URL, client).RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if used != 1 {
		t.Errorf("the given client sent %d requests, want 1", used)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
