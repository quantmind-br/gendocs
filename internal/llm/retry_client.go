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

// RetryConfig holds retry and connection pooling configuration
type RetryConfig struct {
	MaxAttempts       int           // Maximum number of retry attempts
	Multiplier        int           // Exponential backoff multiplier
	MaxWaitPerAttempt time.Duration // Maximum wait time per attempt
	MaxTotalWait      time.Duration // Maximum total wait time

	// Connection pooling settings
	MaxIdleConns        int           // Maximum number of idle connections across all hosts (default: 100)
	MaxIdleConnsPerHost int           // Maximum number of idle connections per host (default: 10)
	IdleConnTimeout     time.Duration // Maximum idle time for a connection (default: 90s)
	TLSHandshakeTimeout time.Duration // Maximum time to wait for TLS handshake (default: 10s)
	ExpectContinueTimeout time.Duration // Maximum time to wait for 100-continue response (default: 1s)
}

// DefaultRetryConfig returns default retry configuration with optimized connection pooling
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:       5,
		Multiplier:        1,
		MaxWaitPerAttempt: 60 * time.Second,
		MaxTotalWait:      300 * time.Second,
		// Connection pooling defaults optimized for LLM APIs
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
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
			Timeout: 180 * time.Second, // Default timeout
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
			Timeout: timeout,
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
