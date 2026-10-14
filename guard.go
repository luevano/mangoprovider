package mangoprovider

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"time"

	"code.dny.dev/ssrf"
	"golang.org/x/net/http/httpproxy"
)

// guardian refuses the IANA special-purpose ranges (loopback, private, link-local, ...) on
// any port: a site's urls (scraped links and images, cdn lists, redirects) can't make
// mangal reach the LAN or the host itself.
var guardian = ssrf.New(ssrf.WithAnyPort())

// GuardClient is a copy of client whose connections only reach public addresses, unless
// Options.AllowPrivateAddresses. The hosts of Options.Mirrors and of the environment's
// proxies are dialed as they are (the user chose them); FlareSolverr has its own client. A
// client with a transport other than *http.Transport can't be guarded and is returned as is,
// a nil one is DefaultOptions' client.
func GuardClient(client *http.Client, options Options) *http.Client {
	if client == nil {
		client = DefaultOptions().HTTPClient
	}
	if options.AllowPrivateAddresses {
		return client
	}
	var transport *http.Transport
	switch t := client.Transport.(type) {
	case nil:
		transport = http.DefaultTransport.(*http.Transport).Clone()
	case *http.Transport:
		transport = t.Clone()
	default:
		return client
	}

	trusted := map[string]bool{}
	for _, site := range options.Mirrors {
		if u, err := url.Parse(site); err == nil && u.Hostname() != "" {
			trusted[u.Hostname()] = true
		}
	}
	if transport.Proxy != nil {
		proxies := httpproxy.FromEnvironment()
		for _, proxy := range []string{proxies.HTTPProxy, proxies.HTTPSProxy} {
			if u, err := url.Parse(proxy); err == nil && u.Hostname() != "" {
				trusted[u.Hostname()] = true
			}
		}
	}

	open := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	guarded := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second, Control: guardian.Safe}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		if host, _, err := net.SplitHostPort(address); err == nil && trusted[host] {
			return open.DialContext(ctx, network, address)
		}
		return guarded.DialContext(ctx, network, address)
	}
	guardedClient := *client
	guardedClient.Transport = transport
	return &guardedClient
}
