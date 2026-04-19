package config

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Client is a thin wrapper to fetch configurations from your central configsvc
type Client struct {
	baseURL *url.URL
	http    *http.Client
}

// NewClient creates a new Config Service client.
// It automatically wraps its internal HTTP transport with OpenTelemetry,
// so any config fetch will automatically be traced locally and map back to Zipkin!
func NewClient(endpoint string, timeout time.Duration, maxRetries int, retryDelay time.Duration) (*Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	// Best-practice: clone the default transport to inherit proxy/TLS settings,
	// but significantly increase the connection pool limits for microservice traffic!
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.MaxIdleConns = 100
	customTransport.MaxIdleConnsPerHost = 100

	return &Client{
		baseURL: u,
		http: &http.Client{
			// Use the provided overarching timeout
			Timeout: timeout,
			// Wrap the open telemetry transport with our custom Retry logic
			Transport: &RetryTransport{
				Base:       otelhttp.NewTransport(customTransport),
				MaxRetries: maxRetries,
				Delay:      retryDelay,
			},
		},
	}, nil
}

// RetryTransport automatically retries failed requests (network errors or 5xx status codes)
type RetryTransport struct {
	Base       http.RoundTripper
	MaxRetries int
	Delay      time.Duration
}

func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	var timer *time.Timer

	// Safeguard to ensure timer is stopped if we exit unexpectedly
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	for i := 0; i <= t.MaxRetries; i++ {
		resp, err = t.Base.RoundTrip(req)

		// Success or client error (4xx) - no need to retry
		if err == nil && resp.StatusCode < http.StatusInternalServerError {
			return resp, nil
		}

		// Don't retry if the original application context was canceled
		if req.Context().Err() != nil {
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			return nil, req.Context().Err()
		}

		// Close body to prevent connection leaks across retries
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}

		if i < t.MaxRetries {
			// Calculate exponential backoff: Delay * 2^i
			backoff := time.Duration(int64(t.Delay) * (1 << i))

			// Cap the maximum backoff to 5 seconds to prevent absurdly long waits
			if backoff > 5*time.Second {
				backoff = 5 * time.Second
			}

			// Apply "Full Jitter": random value between [0, backoff)
			// This completely eliminates the "thundering herd" problem
			jitter := time.Duration(rand.Int64N(int64(backoff) + 1))

			if timer == nil {
				timer = time.NewTimer(jitter)
			} else {
				timer.Reset(jitter)
			}

			select {
			case <-timer.C: // wait before next attempt
			case <-req.Context().Done():
				return nil, req.Context().Err()
			}
		}
	}
	return resp, err
}

// GetConfig fetches the JSON config and decodes it directly into your Go struct.
func (c *Client) GetConfig(ctx context.Context, service string, version string, out interface{}) error {
	path := fmt.Sprintf("/api/v1/configs/%s/versions/%s", url.PathEscape(service), url.PathEscape(version))
	reqURL := c.baseURL.ResolveReference(&url.URL{Path: path})

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return fmt.Errorf("config fetch init error: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("config fetch network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("configuration not found for service: %s, version: %s", service, version)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d when fetching config", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("failed to decode config json: %w", err)
	}

	return nil
}

// ListVersions fetches all available semantic versions for a specific service.
func (c *Client) ListVersions(ctx context.Context, service string) ([]string, error) {
	path := fmt.Sprintf("/api/v1/configs/%s/versions", url.PathEscape(service))
	reqURL := c.baseURL.ResolveReference(&url.URL{Path: path})

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("config versions init error: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("config versions network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d when fetching versions", resp.StatusCode)
	}

	var versions []string
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, fmt.Errorf("failed to decode versions json: %w", err)
	}

	return versions, nil
}
