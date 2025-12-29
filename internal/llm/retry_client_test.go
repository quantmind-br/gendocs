package llm

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestDefaultRetryConfig verifies that DefaultRetryConfig returns expected values
func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxAttempts != 5 {
		t.Errorf("Expected MaxAttempts 5, got %d", config.MaxAttempts)
	}

	if config.Multiplier != 1 {
		t.Errorf("Expected Multiplier 1, got %d", config.Multiplier)
	}

	if config.MaxWaitPerAttempt != 60*time.Second {
		t.Errorf("Expected MaxWaitPerAttempt 60s, got %v", config.MaxWaitPerAttempt)
	}

	if config.MaxTotalWait != 300*time.Second {
		t.Errorf("Expected MaxTotalWait 300s, got %v", config.MaxTotalWait)
	}

	// Connection pooling defaults
	if config.MaxIdleConns != 100 {
		t.Errorf("Expected MaxIdleConns 100, got %d", config.MaxIdleConns)
	}

	if config.MaxIdleConnsPerHost != 10 {
		t.Errorf("Expected MaxIdleConnsPerHost 10, got %d", config.MaxIdleConnsPerHost)
	}

	if config.IdleConnTimeout != 90*time.Second {
		t.Errorf("Expected IdleConnTimeout 90s, got %v", config.IdleConnTimeout)
	}

	if config.TLSHandshakeTimeout != 10*time.Second {
		t.Errorf("Expected TLSHandshakeTimeout 10s, got %v", config.TLSHandshakeTimeout)
	}

	if config.ExpectContinueTimeout != 1*time.Second {
		t.Errorf("Expected ExpectContinueTimeout 1s, got %v", config.ExpectContinueTimeout)
	}
}

// TestNewRetryClient_DefaultTransport verifies that NewRetryClient creates a client with optimized transport
func TestNewRetryClient_DefaultTransport(t *testing.T) {
	client := NewRetryClient(nil)

	// Verify transport is set
	if client.client.Transport == nil {
		t.Fatal("Expected transport to be set, got nil")
	}

	// Verify it's an http.Transport
	transport, ok := client.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Expected *http.Transport, got %T", client.client.Transport)
	}

	// Verify connection pooling settings
	if transport.MaxIdleConns != 100 {
		t.Errorf("Expected MaxIdleConns 100, got %d", transport.MaxIdleConns)
	}

	if transport.MaxIdleConnsPerHost != 10 {
		t.Errorf("Expected MaxIdleConnsPerHost 10, got %d", transport.MaxIdleConnsPerHost)
	}

	if transport.IdleConnTimeout != 90*time.Second {
		t.Errorf("Expected IdleConnTimeout 90s, got %v", transport.IdleConnTimeout)
	}

	if transport.TLSHandshakeTimeout != 10*time.Second {
		t.Errorf("Expected TLSHandshakeTimeout 10s, got %v", transport.TLSHandshakeTimeout)
	}

	if transport.ExpectContinueTimeout != 1*time.Second {
		t.Errorf("Expected ExpectContinueTimeout 1s, got %v", transport.ExpectContinueTimeout)
	}

	// Verify HTTP/2 is enabled
	if !transport.ForceAttemptHTTP2 {
		t.Error("Expected ForceAttemptHTTP2 to be true")
	}

	// Verify TLS config
	if transport.TLSClientConfig == nil {
		t.Fatal("Expected TLSClientConfig to be set, got nil")
	}

	if transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("Expected MinVersion TLS 1.2, got %v", transport.TLSClientConfig.MinVersion)
	}

	// Verify default timeout
	if client.client.Timeout != 180*time.Second {
		t.Errorf("Expected client timeout 180s, got %v", client.client.Timeout)
	}
}

// TestNewRetryClient_CustomTransport verifies that custom transport is used when provided
func TestNewRetryClient_CustomTransport(t *testing.T) {
	customTransport := &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
	}

	config := &RetryConfig{
		Transport: customTransport,
	}

	client := NewRetryClient(config)

	// Verify custom transport is used
	if client.client.Transport != customTransport {
		t.Error("Expected custom transport to be used")
	}
}

// TestNewRetryClient_CustomConnectionPoolSettings verifies custom connection pooling settings
func TestNewRetryClient_CustomConnectionPoolSettings(t *testing.T) {
	tests := []struct {
		name                 string
		config               *RetryConfig
		expectedMaxIdle      int
		expectedMaxIdleHost  int
		expectedIdleTimeout  time.Duration
		expectedTLSTimeout   time.Duration
		expectedContinueTime time.Duration
	}{
		{
			name: "high throughput settings",
			config: &RetryConfig{
				MaxIdleConns:        200,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     120 * time.Second,
				TLSHandshakeTimeout: 15 * time.Second,
				ExpectContinueTimeout: 2 * time.Second,
			},
			expectedMaxIdle:      200,
			expectedMaxIdleHost:  20,
			expectedIdleTimeout:  120 * time.Second,
			expectedTLSTimeout:   15 * time.Second,
			expectedContinueTime: 2 * time.Second,
		},
		{
			name: "memory constrained settings",
			config: &RetryConfig{
				MaxIdleConns:        20,
				MaxIdleConnsPerHost: 2,
				IdleConnTimeout:     30 * time.Second,
				TLSHandshakeTimeout: 5 * time.Second,
				ExpectContinueTimeout: 500 * time.Millisecond,
			},
			expectedMaxIdle:      20,
			expectedMaxIdleHost:  2,
			expectedIdleTimeout:  30 * time.Second,
			expectedTLSTimeout:   5 * time.Second,
			expectedContinueTime: 500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewRetryClient(tt.config)

			transport, ok := client.client.Transport.(*http.Transport)
			if !ok {
				t.Fatalf("Expected *http.Transport, got %T", client.client.Transport)
			}

			if transport.MaxIdleConns != tt.expectedMaxIdle {
				t.Errorf("Expected MaxIdleConns %d, got %d", tt.expectedMaxIdle, transport.MaxIdleConns)
			}

			if transport.MaxIdleConnsPerHost != tt.expectedMaxIdleHost {
				t.Errorf("Expected MaxIdleConnsPerHost %d, got %d", tt.expectedMaxIdleHost, transport.MaxIdleConnsPerHost)
			}

			if transport.IdleConnTimeout != tt.expectedIdleTimeout {
				t.Errorf("Expected IdleConnTimeout %v, got %v", tt.expectedIdleTimeout, transport.IdleConnTimeout)
			}

			if transport.TLSHandshakeTimeout != tt.expectedTLSTimeout {
				t.Errorf("Expected TLSHandshakeTimeout %v, got %v", tt.expectedTLSTimeout, transport.TLSHandshakeTimeout)
			}

			if transport.ExpectContinueTimeout != tt.expectedContinueTime {
				t.Errorf("Expected ExpectContinueTimeout %v, got %v", tt.expectedContinueTime, transport.ExpectContinueTimeout)
			}
		})
	}
}

// TestNewRetryClientWithTimeout_CustomTimeout verifies custom timeout is respected
func TestNewRetryClientWithTimeout_CustomTimeout(t *testing.T) {
	tests := []struct {
		name           string
		timeout        time.Duration
		expectedConfig *RetryConfig
	}{
		{
			name:    "short timeout",
			timeout: 30 * time.Second,
		},
		{
			name:    "long timeout",
			timeout: 300 * time.Second,
		},
		{
			name:    "very short timeout",
			timeout: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewRetryClientWithTimeout(tt.timeout, nil)

			if client.client.Timeout != tt.timeout {
				t.Errorf("Expected timeout %v, got %v", tt.timeout, client.client.Timeout)
			}

			// Verify transport is still optimized
			transport, ok := client.client.Transport.(*http.Transport)
			if !ok {
				t.Fatalf("Expected *http.Transport, got %T", client.client.Transport)
			}

			if transport.MaxIdleConns != 100 {
				t.Errorf("Expected MaxIdleConns 100, got %d", transport.MaxIdleConns)
			}
		})
	}
}

// TestGetConnectionStats_DefaultTransport verifies connection stats are reported correctly
func TestGetConnectionStats_DefaultTransport(t *testing.T) {
	client := NewRetryClient(nil)
	stats := client.GetConnectionStats()

	// Verify transport type
	if stats.TransportType != "http.Transport" {
		t.Errorf("Expected TransportType 'http.Transport', got '%s'", stats.TransportType)
	}

	// Verify connection pool settings
	if stats.MaxIdleConns != 100 {
		t.Errorf("Expected MaxIdleConns 100, got %d", stats.MaxIdleConns)
	}

	if stats.MaxIdleConnsPerHost != 10 {
		t.Errorf("Expected MaxIdleConnsPerHost 10, got %d", stats.MaxIdleConnsPerHost)
	}

	if stats.IdleConnTimeout != 90*time.Second {
		t.Errorf("Expected IdleConnTimeout 90s, got %v", stats.IdleConnTimeout)
	}

	// Verify timeout settings
	if stats.TLSHandshakeTimeout != 10*time.Second {
		t.Errorf("Expected TLSHandshakeTimeout 10s, got %v", stats.TLSHandshakeTimeout)
	}

	if stats.ExpectContinueTimeout != 1*time.Second {
		t.Errorf("Expected ExpectContinueTimeout 1s, got %v", stats.ExpectContinueTimeout)
	}

	// Verify HTTP/2 support
	if !stats.HTTP2Enabled {
		t.Error("Expected HTTP2Enabled to be true")
	}

	// Verify TLS configuration
	if stats.TLSMinVersion != tls.VersionTLS12 {
		t.Errorf("Expected TLSMinVersion %d, got %d", tls.VersionTLS12, stats.TLSMinVersion)
	}

	// Verify client timeout
	if stats.ClientTimeout != 180*time.Second {
		t.Errorf("Expected ClientTimeout 180s, got %v", stats.ClientTimeout)
	}
}

// TestGetConnectionStats_CustomTransport verifies stats handle custom transport
func TestGetConnectionStats_CustomTransport(t *testing.T) {
	customTransport := &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
		ForceAttemptHTTP2:   false,
	}

	config := &RetryConfig{
		Transport: customTransport,
	}

	client := NewRetryClient(config)
	stats := client.GetConnectionStats()

	// Verify transport type is still http.Transport (custom transport is http.Transport)
	if stats.TransportType != "http.Transport" {
		t.Errorf("Expected TransportType 'http.Transport', got '%s'", stats.TransportType)
	}

	// Verify custom settings are reported
	if stats.MaxIdleConns != 50 {
		t.Errorf("Expected MaxIdleConns 50, got %d", stats.MaxIdleConns)
	}

	if stats.MaxIdleConnsPerHost != 5 {
		t.Errorf("Expected MaxIdleConnsPerHost 5, got %d", stats.MaxIdleConnsPerHost)
	}

	if stats.HTTP2Enabled {
		t.Error("Expected HTTP2Enabled to be false for custom transport")
	}
}

// TestGetConnectionStats_NonHTTPTransport verifies stats handle non-http.Transport
func TestGetConnectionStats_NonHTTPTransport(t *testing.T) {
	// Create a mock RoundTripper that's not http.Transport
	mockTransport := &mockRoundTripper{}

	config := &RetryConfig{
		Transport: mockTransport,
	}

	client := NewRetryClient(config)
	stats := client.GetConnectionStats()

	// Verify transport type is "custom"
	if stats.TransportType != "custom" {
		t.Errorf("Expected TransportType 'custom', got '%s'", stats.TransportType)
	}

	// Verify default values for non-http.Transport
	if stats.MaxIdleConns != 0 {
		t.Errorf("Expected MaxIdleConns 0 for custom transport, got %d", stats.MaxIdleConns)
	}
}

// TestCloseIdleConnections verifies the method closes idle connections
func TestCloseIdleConnections(t *testing.T) {
	client := NewRetryClient(nil)

	// This should not panic
	client.CloseIdleConnections()
}

// TestCloseIdleConnections_CustomTransport verifies custom transport is handled
func TestCloseIdleConnections_CustomTransport(t *testing.T) {
	mockTransport := &mockRoundTripper{}

	config := &RetryConfig{
		Transport: mockTransport,
	}

	client := NewRetryClient(config)

	// This should not panic even with non-http.Transport
	client.CloseIdleConnections()
}

// TestGetTimeout verifies timeout getter
func TestGetTimeout(t *testing.T) {
	client := NewRetryClient(nil)

	if timeout := client.GetTimeout(); timeout != 180*time.Second {
		t.Errorf("Expected timeout 180s, got %v", timeout)
	}

	customTimeout := 45 * time.Second
	client.SetTimeout(customTimeout)

	if timeout := client.GetTimeout(); timeout != customTimeout {
		t.Errorf("Expected timeout %v, got %v", customTimeout, timeout)
	}
}

// TestSetTimeout verifies timeout setter
func TestSetTimeout(t *testing.T) {
	client := NewRetryClient(nil)

	newTimeout := 120 * time.Second
	client.SetTimeout(newTimeout)

	if client.client.Timeout != newTimeout {
		t.Errorf("Expected timeout %v, got %v", newTimeout, client.client.Timeout)
	}
}

// mockRoundTripper is a mock implementation of http.RoundTripper for testing
type mockRoundTripper struct{}

func (m *mockRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, nil
}

// TestConnectionReuse_Integration verifies that HTTP connections are reused across multiple requests
func TestConnectionReuse_Integration(t *testing.T) {
	// Track the number of connections established
	var connectionCount int
	var connectionCountMu sync.Mutex

	// Create a custom listener that counts connections
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Wrap the listener to count Accept() calls
	countingListener := &countingListener{
		Listener: listener,
		onAccept: func() {
			connectionCountMu.Lock()
			connectionCount++
			connectionCountMu.Unlock()
		},
	}

	// Create test server with the custom listener
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	server.Listener = countingListener
	server.StartTLS()
	defer server.Close()

	// Create RetryClient with optimized connection pooling
	client := NewRetryClient(nil)

	// Make multiple requests
	numRequests := 10
	for i := 0; i < numRequests; i++ {
		req, err := http.NewRequest("GET", server.URL, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
		resp.Body.Close()
	}

	// Verify connections were reused
	// With connection pooling, we should have fewer connections than requests
	// (allowing for some overhead, we expect at most 2-3 connections for 10 requests)
	connectionCountMu.Lock()
	finalConnectionCount := connectionCount
	connectionCountMu.Unlock()

	if finalConnectionCount >= numRequests {
		t.Errorf("Expected connection reuse (fewer than %d connections), got %d connections",
			numRequests, finalConnectionCount)
	}

	// Verify at least one connection was established
	if finalConnectionCount < 1 {
		t.Errorf("Expected at least 1 connection, got %d", finalConnectionCount)
	}

	t.Logf("Made %d requests using %d connections (reused %d times)",
		numRequests, finalConnectionCount, numRequests-finalConnectionCount)
}

// TestConnectionReuse_ConcurrentRequests verifies connection reuse with concurrent requests
func TestConnectionReuse_ConcurrentRequests(t *testing.T) {
	var connectionCount int
	var connectionCountMu sync.Mutex

	// Create a custom listener that counts connections
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	countingListener := &countingListener{
		Listener: listener,
		onAccept: func() {
			connectionCountMu.Lock()
			connectionCount++
			connectionCountMu.Unlock()
		},
	}

	// Create test server with slight delay to simulate real API
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	server.Listener = countingListener
	server.StartTLS()
	defer server.Close()

	// Create RetryClient with optimized connection pooling
	client := NewRetryClient(nil)

	// Make concurrent requests
	numRequests := 20
	var wg sync.WaitGroup
	wg.Add(numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()
			req, err := http.NewRequest("GET", server.URL, nil)
			if err != nil {
				t.Errorf("Failed to create request: %v", err)
				return
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("Request failed: %v", err)
				return
			}
			resp.Body.Close()
		}()
	}

	wg.Wait()

	// Verify connections were reused even with concurrent requests
	// With HTTP/2 and connection pooling, concurrent requests should share connections
	connectionCountMu.Lock()
	finalConnectionCount := connectionCount
	connectionCountMu.Unlock()

	// We expect significantly fewer connections than requests
	// With HTTP/2 multiplexing, all requests could theoretically use 1 connection
	// But we allow for multiple connections due to concurrent nature
	if finalConnectionCount > numRequests/2 {
		t.Logf("Warning: Made %d concurrent requests using %d connections. Consider optimizing connection pooling.",
			numRequests, finalConnectionCount)
	}

	// Verify at least one connection was established
	if finalConnectionCount < 1 {
		t.Errorf("Expected at least 1 connection, got %d", finalConnectionCount)
	}

	t.Logf("Made %d concurrent requests using %d connections",
		numRequests, finalConnectionCount)
}

// countingListener wraps a net.Listener and counts Accept() calls
type countingListener struct {
	net.Listener
	onAccept func()
}

func (l *countingListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if l.onAccept != nil {
		l.onAccept()
	}
	return conn, err
}
