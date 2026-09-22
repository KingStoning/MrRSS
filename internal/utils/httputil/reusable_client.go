package httputil

import (
	"net/http"
	"sync"
	"time"
)

// ReusableClient keeps a bounded connection pool for a service. Settings changes
// replace the pool immediately without cancelling requests already in flight.
type ReusableClient struct {
	mu       sync.Mutex
	client   *http.Client
	proxyURL string
	insecure bool
}

func (p *ReusableClient) Get(settings ProxySettingsProvider, timeout time.Duration) (*http.Client, error) {
	proxyURL := proxyURLFromSettings(settings)
	insecure := insecureSkipTLSVerifyEnabled()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.client != nil && p.proxyURL == proxyURL && p.insecure == insecure && p.client.Timeout == timeout {
		return p.client, nil
	}
	client, err := CreateHTTPClient(proxyURL, timeout)
	if err != nil {
		return nil, err
	}
	transport := client.Transport.(*http.Transport)
	transport.MaxConnsPerHost = 8
	transport.MaxIdleConnsPerHost = 8
	if p.client != nil {
		p.client.CloseIdleConnections()
	}
	p.client, p.proxyURL, p.insecure = client, proxyURL, insecure
	return client, nil
}
