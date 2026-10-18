package flaresolverr

import (
	"context"
	"errors"
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

func TestRoundTripQueuedHonoursContext(t *testing.T) {
	hang := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-hang
	}))
	defer server.Close()
	defer close(hang)

	transport := NewTransport(server.URL)
	first, _ := http.NewRequest(http.MethodGet, "https://example.com/1", nil)
	go transport.RoundTrip(first)
	time.Sleep(50 * time.Millisecond)

	// queued behind the first one, gives up when its ctx ends
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	queued, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com/2", nil)
	start := time.Now()
	if _, err := transport.RoundTrip(queued); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("wanted the ctx's error, got %v", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Error("the queued request waited past its ctx")
	}

	// less than the second flaresolverr needs left
	ctx, cancel = context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	short, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com/3", nil)
	if _, err := NewTransport(server.URL).RoundTrip(short); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("wanted a deadline error, got %v", err)
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
