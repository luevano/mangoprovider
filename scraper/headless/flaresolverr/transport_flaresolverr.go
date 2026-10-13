package flaresolverr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	mango "github.com/luevano/mangoprovider"
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
	mutex  sync.Mutex
}

// NewTransport creates a new transport for the given Flaresolerr URL.
func NewTransport(url string) *TransportFlaresolverr {
	return &TransportFlaresolverr{
		client: new(http.Client),
		uuid:   uuid.New(),
		url:    url,
	}
}

// RoundTrip makes the request through FlareSolverr.
func (t *TransportFlaresolverr) RoundTrip(r *http.Request) (*http.Response, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// flaresolverr (v3) creates the session on its first request, Close destroys it
	var req []byte
	var err error
	var timeout int
	deadline, ok := r.Context().Deadline()
	if ok {
		timeout = int(deadline.Sub(time.Now()).Milliseconds()) - 1000
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

	return &http.Response{
		StatusCode:    flareResponse.Solution.Status,
		Header:        header,
		Body:          io.NopCloser(strings.NewReader(html)),
		ContentLength: int64(len(html)),
		Request:       r,
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
