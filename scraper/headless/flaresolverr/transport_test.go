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
