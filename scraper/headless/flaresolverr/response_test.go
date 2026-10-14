package flaresolverr

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoundTripDecodesV3Response(t *testing.T) {
	for _, headers := range []string{`{}`, `{"content-type": "text/html; charset=utf-8"}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// shape of a FlareSolverr v3 answer: cookies is a list, header values are strings
			w.Write([]byte(`{"status": "ok", "message": "", "solution": {"url": "https://site", "status": 200,
				"headers": ` + headers + `, "response": "<html>hi</html>", "userAgent": "ua",
				"cookies": [{"name": "cf_clearance", "value": "x", "domain": ".site", "path": "/"}]}}`))
		}))

		req, _ := http.NewRequest(http.MethodGet, "https://site", nil)
		resp, err := NewTransport(server.URL).RoundTrip(req)
		server.Close()
		if err != nil {
			t.Fatalf("headers %s: %v", headers, err)
		}
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "<html>hi</html>" || resp.StatusCode != 200 || resp.Header.Get("Content-Type") == "" {
			t.Errorf("headers %s: got status %d, content type %q, body %q", headers, resp.StatusCode, resp.Header.Get("Content-Type"), body)
		}
	}
}

func TestRoundTripFinalURL(t *testing.T) {
	for solutionURL, want := range map[string]string{
		"https://site/404": "https://site/404",
		"":                 "https://site/manga/1",
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"status": "ok", "solution": {"url": "` + solutionURL + `", "status": 200, "headers": {}, "response": "<html></html>"}}`))
		}))

		req, _ := http.NewRequest(http.MethodGet, "https://site/manga/1", nil)
		resp, err := NewTransport(server.URL).RoundTrip(req)
		server.Close()
		if err != nil {
			t.Fatal(err)
		}
		if resp.Request.URL.String() != want || req.URL.String() != "https://site/manga/1" {
			t.Errorf("solution url %q: got response url %s, request url %s", solutionURL, resp.Request.URL, req.URL)
		}
	}
}
