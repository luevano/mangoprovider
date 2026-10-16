package fetch

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/progress"
	mango "github.com/luevano/mangoprovider"
)

// flaky answers statuses[i] to the i-th request (then 200 "ok"), with Retry-After after if
// set, and records the bodies it got.
func flaky(t *testing.T, after string, statuses ...int) (*httptest.Server, *[]string) {
	t.Helper()
	var mu sync.Mutex
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		n := len(bodies)
		bodies = append(bodies, string(body))
		mu.Unlock()
		if n < len(statuses) {
			if after != "" {
				w.Header().Set("Retry-After", after)
			}
			w.WriteHeader(statuses[n])
			w.Write([]byte("failed " + strconv.Itoa(n)))
			return
		}
		w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)
	return server, &bodies
}

func fastRetry() Retry {
	return Retry{MaxTries: 3, MaxElapsed: 5 * time.Second, InitialInterval: time.Millisecond, MaxInterval: 5 * time.Millisecond}
}

func retryClient(r Retry, log *mango.Logger) *http.Client {
	return &http.Client{Transport: Retries(r, log)(http.DefaultTransport)}
}

func do(t *testing.T, ctx context.Context, client *http.Client, method, url, body string) (int, string, error) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b), nil
}

func TestRetries(t *testing.T) {
	for _, tt := range []struct {
		name     string
		statuses []int
		status   int
		body     string
		requests int
	}{
		{"transient 5xx", []int{503, 502}, 200, "ok", 3},
		{"429", []int{429}, 200, "ok", 2},
		{"out of tries, the last response", []int{500, 500, 500, 500}, 500, "failed 2", 3},
		{"4xx isn't retried", []int{404}, 404, "failed 0", 1},
		{"403 isn't retried", []int{403}, 403, "failed 0", 1},
		{"501 isn't retried", []int{501}, 501, "failed 0", 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server, bodies := flaky(t, "", tt.statuses...)
			status, body, err := do(t, context.Background(), retryClient(fastRetry(), nil), http.MethodGet, server.URL, "")
			if err != nil || status != tt.status || body != tt.body || len(*bodies) != tt.requests {
				t.Errorf("got %d %q %v after %d requests, want %d %q after %d", status, body, err, len(*bodies), tt.status, tt.body, tt.requests)
			}
		})
	}
}

// a POST is sent again with its body
func TestRetriesBody(t *testing.T) {
	server, bodies := flaky(t, "", 503)
	status, _, err := do(t, context.Background(), retryClient(fastRetry(), nil), http.MethodPost, server.URL, "a=1")
	if err != nil || status != 200 || strings.Join(*bodies, ",") != "a=1,a=1" {
		t.Errorf("got %d %v, bodies %q", status, err, *bodies)
	}
}

func TestRetriesRetryAfter(t *testing.T) {
	server, bodies := flaky(t, "1", 429)
	var statuses []string
	ctx := progress.WithReporter(context.Background(), func(s progress.Status) {
		statuses = append(statuses, s.Message)
	})
	log := mango.NewProviderLogger(libmangal.ProviderInfo{ID: "test", Name: "Test"})
	started := time.Now()
	status, _, err := do(t, ctx, retryClient(fastRetry(), log), http.MethodGet, server.URL, "")
	if err != nil || status != 200 || len(*bodies) != 2 {
		t.Fatalf("got %d %v after %d requests", status, err, len(*bodies))
	}
	// Retry-After wins over the 1ms backoff
	if waited := time.Since(started); waited < time.Second {
		t.Errorf("waited %s, want the 1s of Retry-After", waited)
	}
	if len(statuses) != 1 || !strings.Contains(statuses[0], "429 too many requests, retry 1/2 in 1s") {
		t.Errorf("statuses %q", statuses)
	}
}

// a Retry-After past MaxElapsed isn't waited: the 429 is the answer (mango.RateLimitError)
func TestRetriesRetryAfterTooLong(t *testing.T) {
	server, bodies := flaky(t, "120", 429)
	started := time.Now()
	status, _, err := do(t, context.Background(), retryClient(fastRetry(), nil), http.MethodGet, server.URL, "")
	if err != nil || status != 429 || len(*bodies) != 1 || time.Since(started) > time.Second {
		t.Errorf("got %d %v after %d requests in %s", status, err, len(*bodies), time.Since(started))
	}
}

func TestRetriesCancelled(t *testing.T) {
	server, _ := flaky(t, "2", 503, 503)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, _, err := do(t, ctx, retryClient(fastRetry(), nil), http.MethodGet, server.URL, "")
	if err == nil {
		t.Error("want the ctx error")
	}
}

// a challenge is for HeadlessFallback, not retried
func TestRetriesChallenge(t *testing.T) {
	server := site(t)
	var requests int
	client := &http.Client{Transport: Retries(fastRetry(), nil)(RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		return http.DefaultTransport.RoundTrip(r)
	}))}
	status, _, err := do(t, context.Background(), client, http.MethodGet, server.URL+"/mitigated", "")
	if err != nil || status != 503 || requests != 1 {
		t.Errorf("got %d %v after %d requests", status, err, requests)
	}
}

// NewClient puts the retries outside the limiter: every try waits for it
func TestClientRetry(t *testing.T) {
	server, bodies := flaky(t, "", 500)
	r := fastRetry()
	c, err := NewClient(nil, mango.DefaultHeadlessOptions(), nil, Config{Retry: &r})
	if err != nil {
		t.Fatal(err)
	}
	status, _, err := do(t, context.Background(), c.Client, http.MethodGet, server.URL, "")
	if err != nil || status != 200 || len(*bodies) != 2 {
		t.Errorf("got %d %v after %d requests", status, err, len(*bodies))
	}
}

func TestRetriesServerErrors(t *testing.T) {
	server, bodies := flaky(t, "", 429, 503)
	r := fastRetry()
	r.Statuses = ServerErrors
	status, _, err := do(t, context.Background(), retryClient(r, nil), http.MethodGet, server.URL, "")
	if err != nil || status != 429 || len(*bodies) != 1 {
		t.Errorf("429: got %d %v after %d requests", status, err, len(*bodies))
	}
	status, _, err = do(t, context.Background(), retryClient(r, nil), http.MethodGet, server.URL, "")
	if err != nil || status != 200 || len(*bodies) != 3 {
		t.Errorf("503: got %d %v after %d requests", status, err, len(*bodies))
	}
}

func TestRetryClient(t *testing.T) {
	server, bodies := flaky(t, "", 502)
	base := &http.Client{Timeout: time.Minute}
	client := RetryClient(base, fastRetry(), nil, nil)
	if base.Transport != nil || client.Timeout != time.Minute {
		t.Fatal("base changed or not copied")
	}
	status, _, err := do(t, context.Background(), client, http.MethodGet, server.URL, "")
	if err != nil || status != 200 || len(*bodies) != 2 {
		t.Errorf("got %d %v after %d requests", status, err, len(*bodies))
	}
}
