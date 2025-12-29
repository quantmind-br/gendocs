package llm

import (
	"fmt"
	"net/http"
	"time"
)

// Example_defaultConfiguration demonstrates using the RetryClient with
// default optimized connection pooling settings.
//
// This is the recommended approach for most use cases. The defaults are
// optimized for LLM API usage and provide good performance out of the box.
func Example_defaultConfiguration() {
	// Create a client with default configuration
	client := NewRetryClient(nil)

	// The client is now ready to use with optimized connection pooling:
	// - MaxIdleConns: 100 (global connection pool)
	// - MaxIdleConnsPerHost: 10 (per-host pool, e.g., api.openai.com)
	// - IdleConnTimeout: 90s (connections idle for 90s are kept alive)
	// - TLSHandshakeTimeout: 10s
	// - ExpectContinueTimeout: 1s
	// - HTTP/2 enabled
	// - TLS 1.2 minimum

	// You can verify the configuration
	stats := client.GetConnectionStats()
	fmt.Printf("Transport type: %s\n", stats.TransportType)
	fmt.Printf("HTTP/2 enabled: %v\n", stats.HTTP2Enabled)
	fmt.Printf("Max idle connections: %d\n", stats.MaxIdleConns)
	fmt.Printf("Max idle connections per host: %d\n", stats.MaxIdleConnsPerHost)
	fmt.Printf("Idle connection timeout: %v\n", stats.IdleConnTimeout)

	// Use the client for API calls
	// req, _ := http.NewRequest("GET", "https://api.openai.com/v1/models", nil)
	// resp, err := client.Do(req)
	// _ = resp // Handle response

	_ = client // In real usage, you would make API calls here
}

// Example_customHighThroughput demonstrates custom connection pooling
// configuration for high-throughput scenarios.
//
// Use this configuration when you need to handle many concurrent requests
// to LLM APIs, such as in a server environment handling multiple users.
func Example_customHighThroughput() {
	config := &RetryConfig{
		// Retry settings
		MaxAttempts:       5,
		Multiplier:        1,
		MaxWaitPerAttempt: 60 * time.Second,
		MaxTotalWait:      300 * time.Second,

		// Connection pooling for high throughput
		// Increase pool sizes to handle more concurrent connections
		MaxIdleConns:        500,  // Larger global pool (default: 100)
		MaxIdleConnsPerHost: 50,   // More connections per host (default: 10)
		IdleConnTimeout:     120 * time.Second, // Keep connections alive longer (default: 90s)

		// Timeout settings
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := NewRetryClient(config)

	stats := client.GetConnectionStats()
	fmt.Printf("High-throughput configuration:\n")
	fmt.Printf("  Max idle connections: %d\n", stats.MaxIdleConns)
	fmt.Printf("  Max idle connections per host: %d\n", stats.MaxIdleConnsPerHost)
	fmt.Printf("  Idle connection timeout: %v\n", stats.IdleConnTimeout)

	_ = client // In real usage, you would make API calls here
}

// Example_customMemoryConstrained demonstrates custom connection pooling
// configuration for memory-constrained environments.
//
// Use this configuration when running in environments with limited memory,
// such as AWS Lambda, Cloud Functions, or embedded devices.
func Example_customMemoryConstrained() {
	config := &RetryConfig{
		// Retry settings
		MaxAttempts:       3, // Fewer retries to save resources
		Multiplier:        1,
		MaxWaitPerAttempt: 30 * time.Second,
		MaxTotalWait:      60 * time.Second,

		// Connection pooling for memory efficiency
		// Reduce pool sizes to minimize memory footprint
		MaxIdleConns:        20,  // Smaller global pool (default: 100)
		MaxIdleConnsPerHost: 5,   // Fewer connections per host (default: 10)
		IdleConnTimeout:     30 * time.Second, // Shorter timeout to free resources faster (default: 90s)

		// Timeout settings
		TLSHandshakeTimeout:   5 * time.Second,  // Faster timeout
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := NewRetryClient(config)

	stats := client.GetConnectionStats()
	fmt.Printf("Memory-constrained configuration:\n")
	fmt.Printf("  Max idle connections: %d\n", stats.MaxIdleConns)
	fmt.Printf("  Max idle connections per host: %d\n", stats.MaxIdleConnsPerHost)
	fmt.Printf("  Idle connection timeout: %v\n", stats.IdleConnTimeout)

	_ = client // In real usage, you would make API calls here
}

// Example_customTimeout demonstrates creating a RetryClient with a custom
// timeout while maintaining optimized connection pooling.
func Example_customTimeout() {
	// Create a client with a 2-minute timeout
	// Connection pooling settings will use optimized defaults
	client := NewRetryClientWithTimeout(2*time.Minute, nil)

	fmt.Printf("Client timeout: %v\n", client.GetTimeout())

	_ = client // In real usage, you would make API calls here
}

// Example_customTransport demonstrates providing a completely custom HTTP transport.
//
// Use this when you need advanced features like:
// - Custom proxy configuration
// - Custom TLS configuration
// - Custom dialer (e.g., for SOCKS proxy)
// - Advanced connection management
func Example_customTransport() {
	// Create a custom transport with specific requirements
	customTransport := &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		// Add your custom configuration here
		// For example, proxy, custom dialer, etc.
	}

	config := &RetryConfig{
		// Retry settings
		MaxAttempts:       5,
		Multiplier:        1,
		MaxWaitPerAttempt: 60 * time.Second,
		MaxTotalWait:      300 * time.Second,

		// Use custom transport - connection pooling fields above will be ignored
		Transport: customTransport,
	}

	client := NewRetryClient(config)

	stats := client.GetConnectionStats()
	fmt.Printf("Custom transport type: %s\n", stats.TransportType)
	fmt.Printf("Max idle connections from custom transport: %d\n", stats.MaxIdleConns)

	_ = client // In real usage, you would make API calls here
}

// Example_connectionStats demonstrates how to retrieve connection pool
// statistics for monitoring and debugging.
func Example_connectionStats() {
	client := NewRetryClient(nil)

	stats := client.GetConnectionStats()

	fmt.Printf("=== Connection Pool Statistics ===\n")
	fmt.Printf("Transport type: %s\n", stats.TransportType)
	fmt.Printf("HTTP/2 enabled: %v\n", stats.HTTP2Enabled)
	fmt.Printf("TLS minimum version: %d\n", stats.TLSMinVersion)
	fmt.Printf("\nConnection Pool Configuration:\n")
	fmt.Printf("  Max idle connections (global): %d\n", stats.MaxIdleConns)
	fmt.Printf("  Max idle connections per host: %d\n", stats.MaxIdleConnsPerHost)
	fmt.Printf("  Idle connection timeout: %v\n", stats.IdleConnTimeout)
	fmt.Printf("\nTimeout Configuration:\n")
	fmt.Printf("  TLS handshake timeout: %v\n", stats.TLSHandshakeTimeout)
	fmt.Printf("  Expect continue timeout: %v\n", stats.ExpectContinueTimeout)
	fmt.Printf("  Client timeout: %v\n", stats.ClientTimeout)

	_ = client // In real usage, you would make API calls here
}

// Example_closeIdleConnections demonstrates how to manually close idle
// connections to free up resources.
//
// This can be useful in long-running applications when you know the client
// will not be used for a while, or when shutting down gracefully.
func Example_closeIdleConnections() {
	client := NewRetryClient(nil)

	// ... use the client for API calls ...

	// When done, explicitly close idle connections to free resources
	client.CloseIdleConnections()

	// The client can still be used after closing idle connections
	// New connections will be established as needed

	_ = client
}

// Example_multipleProviders demonstrates using connection pooling with
// multiple LLM API providers.
//
// The global connection pool (MaxIdleConns) manages connections across all
// providers, while MaxIdleConnsPerHost limits connections per provider.
func Example_multipleProviders() {
	config := &RetryConfig{
		// Retry settings
		MaxAttempts:       5,
		Multiplier:        1,
		MaxWaitPerAttempt: 60 * time.Second,
		MaxTotalWait:      300 * time.Second,

		// Connection pooling for multiple providers
		MaxIdleConns:        200, // Total pool for all providers
		MaxIdleConnsPerHost: 10,  // Per-provider limit (e.g., api.openai.com, api.anthropic.com)
		IdleConnTimeout:     90 * time.Second,

		// Timeout settings
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := NewRetryClient(config)

	// This client can efficiently handle requests to multiple LLM providers:
	// - api.openai.com: up to 10 idle connections
	// - api.anthropic.com: up to 10 idle connections
	// - generativelanguage.googleapis.com: up to 10 idle connections
	// All while keeping total idle connections under 200

	stats := client.GetConnectionStats()
	fmt.Printf("Multiple provider configuration:\n")
	fmt.Printf("  Total max idle connections: %d\n", stats.MaxIdleConns)
	fmt.Printf("  Max idle connections per provider: %d\n", stats.MaxIdleConnsPerHost)

	_ = client
}
