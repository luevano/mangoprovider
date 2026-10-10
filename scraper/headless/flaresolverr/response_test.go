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
