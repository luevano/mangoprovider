package flaresolverr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

var _ http.RoundTripper = (*TransportFlaresolverr)(nil)

// TransportFlaresolverr implementation of Transport used for colly.
type TransportFlaresolverr struct {
	client      *http.Client
	uuid        uuid.UUID
	url         string
	mutex       sync.Mutex
}

// NewTransport creates a new transport for the given Flaresolerr URL.
func NewTransport(url string) *TransportFlaresolverr {
	return &TransportFlaresolverr{
		client: new(http.Client),
		uuid:   uuid.New(),
		url:    url,
	}
}

// RoundTrip gets called on each colly request.
func (t *TransportFlaresolverr) RoundTrip(r *http.Request) (*http.Response, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	var req []byte
	var err error
	var timeout int
	deadline, ok := r.Context().Deadline()
	if ok {
		timeout = int(deadline.Sub(time.Now()).Milliseconds()) - 1000
	} else {
		timeout = 30000
	}

	switch r.Method {
	case "GET":
		req, err = json.Marshal(request{
			Cmd:        "request.get",
			URL:        r.URL.String(),
			MaxTimeout: timeout,
			Session:    t.uuid.String(),
		})
	case "POST":
		content, err := io.ReadAll(r.Body)
		if err != nil {
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
		return nil, errors.New("only supports GET and POST methods")
	}
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(r.Context(), "POST", t.url, bytes.NewBuffer(req))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var flareResponse response
	err = json.Unmarshal(body, &flareResponse)
	if err != nil {
		return nil, err
	}

	if flareResponse.Status == "error" {
		return nil, errors.New(flareResponse.Message)
	}

	response := &http.Response{
		StatusCode: flareResponse.Solution.Status,
		Header:     flareResponse.Solution.Headers,
		Body:       newContainer(&flareResponse),
		Request:    r,
	}
	if response.Header.Get("Content-Type") == "" {
		response.Header.Set("Content-Type", "text/html")
		response.Header.Set("Content-Length", strconv.Itoa(len(body)))
	}

	return response, nil
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
	request, err := http.NewRequestWithContext(context.Background(), "POST", t.url, bytes.NewBuffer(req))
	request.Header.Set("Content-Type", "application/json")
	if err != nil {
		return err
	}
	resp, err := t.client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
