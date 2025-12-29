package llm

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

// RetryConfig holds retry and connection pooling configuration.
//
// The connection pooling settings are optimized for LLM API usage, where
// requests to the same API endpoint are common. Keeping connections open
// and reusing them significantly reduces latency by eliminating the need
// for repeated TCP and TLS handshakes.
type RetryConfig struct {
	// Retry settings
	MaxAttempts       int           // Maximum number of retry attempts (default: 5)
	Multiplier        int           // Exponential backoff multiplier in seconds (default: 1)
	MaxWaitPerAttempt time.Duration // Maximum wait time per retry attempt (default: 60s)
	MaxTotalWait      time.Duration // Maximum total wait time across all retries (default: 300s)

	// Connection Pooling Settings
	//
	// These settings control how HTTP connections are managed and reused.
	// Proper connection pooling is critical for LLM API performance because:
	// - It reduces latency by reusing existing connections
	// - It minimizes TCP/TLS handshake overhead
	// - It improves throughput by maintaining connection pools to frequently-used hosts
	//
	// MaxIdleConns controls the maximum number of idle connections across ALL hosts.
	// This is the global pool size. Increasing this allows more connections to be kept
	// open simultaneously, useful when making requests to multiple different LLM providers.
	// Higher values = more memory usage but better performance for concurrent requests.
	// Default: 100, Range: 10-1000 recommended
	MaxIdleConns int

	// MaxIdleConnsPerHost controls the maximum number of idle connections PER HOST.
	// This is the pool size for each unique hostname (e.g., "api.openai.com").
	// LLM APIs typically benefit from higher values here because:
	// - Multiple concurrent requests to the same API endpoint are common
	// - Keeping multiple connections open allows for better request pipelining
	// - It prevents connection churn under high load
	// Default: 10, Range: 5-100 recommended. Set to 2-3x your expected concurrent request rate.
	MaxIdleConnsPerHost int

	// IdleConnTimeout controls how long an idle connection remains in the pool before
	// being closed. Idle connections that exceed this duration are pruned to free resources.
	// For LLM APIs with intermittent but bursty traffic, a longer timeout is beneficial
	// because it keeps connections available between bursts of activity.
	// Default: 90s, Range: 30s-300s recommended
	IdleConnTimeout time.Duration

	// TLSHandshakeTimeout specifies the maximum time to wait for a TLS handshake to complete.
	// TLS handshakes establish the secure connection and happen on new connections or when
	// a connection is being reused after a long idle period.
	// Default: 10s, Range: 5s-30s recommended
	TLSHandshakeTimeout time.Duration

	// ExpectContinueTimeout specifies the maximum time to wait for a server's "100 Continue"
	// response when sending a request with an Expect: 100-continue header.
	// This is an optimization for requests with large bodies (common in LLM APIs).
	// The timeout allows the server to indicate whether it will accept the request body
	// before the client sends it, saving bandwidth if the request will be rejected.
	// Default: 1s, Range: 1s-5s recommended
	ExpectContinueTimeout time.Duration

	// Transport allows providing a custom HTTP transport.
	//
	// If nil, a transport with optimized connection pooling settings will be created
	// using the fields above. This is recommended for most use cases.
	//
	// If set, the custom transport will be used directly and the connection pooling
	// fields above (MaxIdleConns, MaxIdleConnsPerHost, etc.) will be ignored.
	// Use this when you need complete control over HTTP transport behavior,
	// such as custom proxies, authentication, or advanced connection management.
	Transport http.RoundTripper
}

// DefaultRetryConfig returns default retry configuration with optimized connection pooling
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:           5,
		Multiplier:            1,
		MaxWaitPerAttempt:     60 * time.Second,
		MaxTotalWait:          300 * time.Second,
		// Connection pooling defaults optimized for LLM APIs
		MaxIdleConns:           100,
		MaxIdleConnsPerHost:    10,
		IdleConnTimeout:        90 * time.Second,
		TLSHandshakeTimeout:    10 * time.Second,
		ExpectContinueTimeout:  1 * time.Second,
	}
}

// getTransport returns the appropriate HTTP transport for the given config
// If a custom transport is provided in the config, it will be used
// Otherwise, an optimized transport will be created using the config's connection pooling settings
func getTransport(config *RetryConfig) http.RoundTripper {
	if config.Transport != nil {
		// Use custom transport provided by user
		return config.Transport
	}
	// Create optimized transport with config settings
	return createOptimizedTransport(config)
}

// createOptimizedTransport creates an http.Transport with optimal settings for LLM API calls
// It configures connection pooling, timeouts, and HTTP/2 support for improved performance
func createOptimizedTransport(config *RetryConfig) *http.Transport {
	transport := &http.Transport{
		// Connection pooling settings
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxIdleConnsPerHost,
		IdleConnTimeout:     config.IdleConnTimeout,

		// Timeout settings
		TLSHandshakeTimeout:   config.TLSHandshakeTimeout,
		ExpectContinueTimeout: config.ExpectContinueTimeout,

		// Force attempt HTTP/2 for HTTPS connections
		// Note: Go's http2.ConfigureTransport will enable HTTP/2 if supported
		ForceAttemptHTTP2: true,
	}

	// Configure TLS settings for optimal performance
	transport.TLSClientConfig = &tls.Config{
		// Use reasonable defaults for TLS
		MinVersion: tls.VersionTLS12,
		// Enable HTTP/2 properly (will be configured by http2.ConfigureTransport if available)
	}

	return transport
}

// ConnectionStats represents connection pool statistics and configuration
type ConnectionStats struct {
	// Transport type
	TransportType string // "http.Transport", "custom", or "unknown"

	// Connection pool configuration
	MaxIdleConns        int           // Maximum idle connections across all hosts
	MaxIdleConnsPerHost int           // Maximum idle connections per host
	IdleConnTimeout     time.Duration // Maximum idle time for a connection

	// Timeout configuration
	TLSHandshakeTimeout   time.Duration // TLS handshake timeout
	ExpectContinueTimeout time.Duration // Expect continue timeout

	// HTTP/2 support
	HTTP2Enabled bool // Whether HTTP/2 is enabled

	// TLS configuration
	TLSMinVersion uint16 // Minimum TLS version

	// Client configuration
	ClientTimeout time.Duration // Client timeout
}

// RetryClient wraps http.Client with retry logic
type RetryClient struct {
	client *http.Client
	config *RetryConfig
}

// NewRetryClient creates a new retry client
func NewRetryClient(config *RetryConfig) *RetryClient {
	if config == nil {
		config = DefaultRetryConfig()
	}

	return &RetryClient{
		client: &http.Client{
			Timeout:   180 * time.Second, // Default timeout
			Transport: getTransport(config),
		},
		config: config,
	}
}

// NewRetryClientWithTimeout creates a retry client with custom timeout
func NewRetryClientWithTimeout(timeout time.Duration, config *RetryConfig) *RetryClient {
	if config == nil {
		config = DefaultRetryConfig()
	}

	return &RetryClient{
		client: &http.Client{
			Timeout:   timeout,
			Transport: getTransport(config),
		},
		config: config,
	}
}

// Do executes an HTTP request with retry logic
func (rc *RetryClient) Do(req *http.Request) (*http.Response, error) {
	return rc.DoWithContext(req.Context(), req)
}

// DoWithContext executes an HTTP request with retry logic and context
func (rc *RetryClient) DoWithContext(ctx context.Context, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	// Read request body once and store for potential retries
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, err = io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
	}

	totalStartTime := time.Now()

	for attempt := 0; attempt < rc.config.MaxAttempts; attempt++ {
		// Clone the request for each attempt
		reqClone := req.Clone(ctx)

		// Restore body for this attempt
		if len(bodyBytes) > 0 {
			reqClone.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			reqClone.ContentLength = int64(len(bodyBytes))
		}

		resp, err = rc.client.Do(reqClone)

		// Check if we should NOT retry
		if err == nil && resp != nil {
			// Success on 2xx and 3xx
			// Also retry on 429 (Too Many Requests) and 5xx
			if resp.StatusCode < 500 && resp.StatusCode != 429 {
				return resp, nil
			}

			// For 4xx errors (except 429), don't retry
			if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
				return resp, nil // Return the error response to caller
			}
		}

		// Calculate wait time with exponential backoff
		waitTime := rc.calculateWaitTime(attempt)

		// Check if we've exceeded max total wait time
		if time.Since(totalStartTime)+waitTime > rc.config.MaxTotalWait {
			break
		}

		// Wait before retry (but not after the last attempt)
		if attempt < rc.config.MaxAttempts-1 {
			select {
			case <-time.After(waitTime):
				// Continue to next attempt
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	// All retries exhausted
	if err != nil {
		return nil, fmt.Errorf("request failed after %d attempts: %w", rc.config.MaxAttempts, err)
	}

	if resp != nil {
		return nil, fmt.Errorf("request failed with status %d after %d attempts", resp.StatusCode, rc.config.MaxAttempts)
	}

	return nil, fmt.Errorf("request failed after %d attempts", rc.config.MaxAttempts)
}

// calculateWaitTime calculates wait time using exponential backoff
func (rc *RetryClient) calculateWaitTime(attempt int) time.Duration {
	// Exponential backoff: 2^attempt * multiplier seconds
	baseWait := time.Duration(math.Pow(2, float64(attempt))) * time.Duration(rc.config.Multiplier) * time.Second

	// Cap at max wait per attempt
	if baseWait > rc.config.MaxWaitPerAttempt {
		baseWait = rc.config.MaxWaitPerAttempt
	}

	return baseWait
}

// SetTimeout updates the client timeout
func (rc *RetryClient) SetTimeout(timeout time.Duration) {
	rc.client.Timeout = timeout
}

// GetTimeout returns the current client timeout
func (rc *RetryClient) GetTimeout() time.Duration {
	return rc.client.Timeout
}

// GetConnectionStats returns connection pool statistics and configuration
// This is useful for debugging, monitoring, and verifying connection pooling settings
func (rc *RetryClient) GetConnectionStats() ConnectionStats {
	stats := ConnectionStats{
		ClientTimeout: rc.client.Timeout,
	}

	// Use type assertion to check if we have an http.Transport
	if transport, ok := rc.client.Transport.(*http.Transport); ok {
		stats.TransportType = "http.Transport"
		stats.MaxIdleConns = transport.MaxIdleConns
		stats.MaxIdleConnsPerHost = transport.MaxIdleConnsPerHost
		stats.IdleConnTimeout = transport.IdleConnTimeout
		stats.TLSHandshakeTimeout = transport.TLSHandshakeTimeout
		stats.ExpectContinueTimeout = transport.ExpectContinueTimeout
		stats.HTTP2Enabled = transport.ForceAttemptHTTP2

		// Get TLS configuration if available
		if transport.TLSClientConfig != nil {
			stats.TLSMinVersion = transport.TLSClientConfig.MinVersion
		}
	} else if rc.client.Transport != nil {
		stats.TransportType = "custom"
	} else {
		stats.TransportType = "unknown"
	}

	return stats
}

// CloseIdleConnections closes any idle connections in the transport's connection pool
// This can be useful to free up resources when the client will no longer be used
func (rc *RetryClient) CloseIdleConnections() {
	if transport, ok := rc.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}
