package flaresolverr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	mango "github.com/luevano/mangoprovider"
	"golang.org/x/sync/semaphore"
)

var (
	// defaultMaxTimeout: flaresolverr's maxTimeout when the request has no deadline.
	defaultMaxTimeout = 30 * time.Second
	// requestMargin: How long past maxTimeout to wait for flaresolverr's answer.
	requestMargin = 30 * time.Second
)

var _ http.RoundTripper = (*TransportFlaresolverr)(nil)

// TransportFlaresolverr is an http.RoundTripper that makes the requests through FlareSolverr.
type TransportFlaresolverr struct {
	client *http.Client
	uuid   uuid.UUID
	url    string
	// one request at a time, waited for until the request's ctx ends
	slot *semaphore.Weighted
}

// NewTransport creates a new transport for the given Flaresolerr URL.
func NewTransport(url string) *TransportFlaresolverr {
	return NewTransportClient(url, nil)
}

// NewTransportClient is NewTransport calling FlareSolverr with client, a new one if nil. The
// client shouldn't have a timeout shorter than the solves (each request has its own).
func NewTransportClient(url string, client *http.Client) *TransportFlaresolverr {
	if client == nil {
		client = new(http.Client)
	}
	return &TransportFlaresolverr{
		client: client,
		uuid:   uuid.New(),
		url:    url,
		slot:   semaphore.NewWeighted(1),
	}
}

// RoundTrip makes the request through FlareSolverr.
func (t *TransportFlaresolverr) RoundTrip(r *http.Request) (*http.Response, error) {
	if err := t.slot.Acquire(r.Context(), 1); err != nil {
		return nil, err
	}
	defer t.slot.Release(1)

	// flaresolverr (v3) creates the session on its first request, Close destroys it
	var req []byte
	var err error
	var timeout int
	deadline, ok := r.Context().Deadline()
	if ok {
		timeout = int(time.Until(deadline).Milliseconds()) - 1000
		if timeout <= 0 {
			return nil, fmt.Errorf("flaresolverr: no time left to solve %s: %w", r.URL, context.DeadlineExceeded)
		}
	} else {
		timeout = int(defaultMaxTimeout.Milliseconds())
	}

	switch r.Method {
	case http.MethodGet:
		req, err = json.Marshal(request{
			Cmd:        "request.get",
			URL:        r.URL.String(),
			MaxTimeout: timeout,
			Session:    t.uuid.String(),
		})
	case http.MethodPost:
		var content []byte
		if content, err = io.ReadAll(r.Body); err != nil {
			return nil, err
		}
		req, err = json.Marshal(request{
			Cmd:        "request.post",
			URL:        r.URL.String(),
			MaxTimeout: timeout,
			Session:    t.uuid.String(),
			PostData:   string(content),
		})
	default:
		return nil, fmt.Errorf("flaresolverr: %s not supported, only GET and POST", r.Method)
	}
	if err != nil {
		return nil, err
	}

	// flaresolverr answers within maxTimeout, the margin covers its own overhead
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeout)*time.Millisecond+requestMargin)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, bytes.NewBuffer(req))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := mango.ReadBody(resp.Body)
	if err != nil {
		return nil, err
	}

	var flareResponse response
	err = json.Unmarshal(body, &flareResponse)
	if err != nil {
		return nil, err
	}

	if flareResponse.Status == "error" {
		return nil, fmt.Errorf("flaresolverr: %s", flareResponse.Message)
	}

	html := flareResponse.Solution.Response
	header := http.Header{}
	for k, v := range flareResponse.Solution.Headers {
		header.Set(k, v)
	}
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "text/html")
	}
	header.Set("Content-Length", strconv.Itoa(len(html)))

	// where the browser ended up, so redirects are seen like with a direct request
	final := r
	if u, err := url.Parse(flareResponse.Solution.URL); err == nil && u.IsAbs() {
		final = r.Clone(r.Context())
		final.URL = u
		final.Host = u.Host
	}

	return &http.Response{
		StatusCode:    flareResponse.Solution.Status,
		Header:        header,
		Body:          io.NopCloser(strings.NewReader(html)),
		ContentLength: int64(len(html)),
		Request:       final,
	}, nil
}

func (t *TransportFlaresolverr) Close() error {
	if t.uuid == uuid.Nil {
		return nil
	}
	req, err := json.Marshal(request{
		Cmd:     "sessions.destroy",
		Session: t.uuid.String(),
	})
	if err != nil {
		return err
	}
	// no caller context on close, just don't hang if flaresolverr doesn't answer
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, bytes.NewBuffer(req))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	resp, err := t.client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
