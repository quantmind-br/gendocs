package llm

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// createUnoptimizedTransport creates a basic http.Transport without connection pooling optimization
// This simulates the old behavior before connection pooling was implemented
func createUnoptimizedTransport() *http.Transport {
	return &http.Transport{
		// No connection pooling configuration - uses Go's defaults
		// Default MaxIdleConns: 100 (but may not be optimal)
		// Default MaxIdleConnsPerHost: 2 (too low for LLM APIs)
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // For test servers
		},
	}
}

// BenchmarkSequentialRequests_Optimized benchmarks sequential requests with optimized connection pooling
func BenchmarkSequentialRequests_Optimized(b *testing.B) {
	// Create test server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	// Create client with optimized transport
	client := NewRetryClient(nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest("GET", server.URL, nil)
		if err != nil {
			b.Fatalf("Failed to create request: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
		resp.Body.Close()
	}
}

// BenchmarkSequentialRequests_Unoptimized benchmarks sequential requests without optimized connection pooling
func BenchmarkSequentialRequests_Unoptimized(b *testing.B) {
	// Create test server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	// Create client with unoptimized transport (simulating old behavior)
	httpClient := &http.Client{
		Timeout:   180 * time.Second,
		Transport: createUnoptimizedTransport(),
	}
	client := &RetryClient{
		client: httpClient,
		config: DefaultRetryConfig(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest("GET", server.URL, nil)
		if err != nil {
			b.Fatalf("Failed to create request: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
		resp.Body.Close()
	}
}

// BenchmarkConcurrentRequests_Optimized benchmarks concurrent requests with optimized connection pooling
func BenchmarkConcurrentRequests_Optimized(b *testing.B) {
	// Create test server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate API latency
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	// Create client with optimized transport
	client := NewRetryClient(nil)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, err := http.NewRequest("GET", server.URL, nil)
			if err != nil {
				b.Fatalf("Failed to create request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				b.Fatalf("Request failed: %v", err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkConcurrentRequests_Unoptimized benchmarks concurrent requests without optimized connection pooling
func BenchmarkConcurrentRequests_Unoptimized(b *testing.B) {
	// Create test server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate API latency
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	// Create client with unoptimized transport
	httpClient := &http.Client{
		Timeout:   180 * time.Second,
		Transport: createUnoptimizedTransport(),
	}
	client := &RetryClient{
		client: httpClient,
		config: DefaultRetryConfig(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, err := http.NewRequest("GET", server.URL, nil)
			if err != nil {
				b.Fatalf("Failed to create request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				b.Fatalf("Request failed: %v", err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkConcurrentRequests_10 benchmarks concurrent requests with 10 goroutines
func BenchmarkConcurrentRequests_10_Optimized(b *testing.B) {
	benchmarkConcurrentRequests(b, 10, true)
}

// BenchmarkConcurrentRequests_50 benchmarks concurrent requests with 50 goroutines
func BenchmarkConcurrentRequests_50_Optimized(b *testing.B) {
	benchmarkConcurrentRequests(b, 50, true)
}

// BenchmarkConcurrentRequests_100 benchmarks concurrent requests with 100 goroutines
func BenchmarkConcurrentRequests_100_Optimized(b *testing.B) {
	benchmarkConcurrentRequests(b, 100, true)
}

// BenchmarkConcurrentRequests_10_Unoptimized benchmarks concurrent requests with 10 goroutines (unoptimized)
func BenchmarkConcurrentRequests_10_Unoptimized(b *testing.B) {
	benchmarkConcurrentRequests(b, 10, false)
}

// BenchmarkConcurrentRequests_50_Unoptimized benchmarks concurrent requests with 50 goroutines (unoptimized)
func BenchmarkConcurrentRequests_50_Unoptimized(b *testing.B) {
	benchmarkConcurrentRequests(b, 50, false)
}

// BenchmarkConcurrentRequests_100_Unoptimized benchmarks concurrent requests with 100 goroutines (unoptimized)
func BenchmarkConcurrentRequests_100_Unoptimized(b *testing.B) {
	benchmarkConcurrentRequests(b, 100, false)
}

// benchmarkConcurrentRequests is a helper function for concurrent request benchmarks
func benchmarkConcurrentRequests(b *testing.B, numGoroutines int, optimized bool) {
	// Create test server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate API latency
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	var client *RetryClient
	if optimized {
		client = NewRetryClient(nil)
	} else {
		httpClient := &http.Client{
			Timeout:   180 * time.Second,
			Transport: createUnoptimizedTransport(),
		}
		client = &RetryClient{
			client: httpClient,
			config: DefaultRetryConfig(),
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Run b.N iterations, each with numGoroutines concurrent requests
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for j := 0; j < numGoroutines; j++ {
			go func() {
				defer wg.Done()
				req, err := http.NewRequest("GET", server.URL, nil)
				if err != nil {
					b.Fatalf("Failed to create request: %v", err)
				}

				resp, err := client.Do(req)
				if err != nil {
					b.Fatalf("Request failed: %v", err)
				}
				resp.Body.Close()
			}()
		}

		wg.Wait()
	}
}

// BenchmarkConnectionPoolSettings benchmarks different connection pool configurations
func BenchmarkConnectionPoolSettings_HighThroughput(b *testing.B) {
	config := &RetryConfig{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     120 * time.Second,
		TLSHandshakeTimeout: 15 * time.Second,
		ExpectContinueTimeout: 2 * time.Second,
	}

	benchmarkWithConfig(b, config)
}

// BenchmarkConnectionPoolSettings_MemoryConstrained benchmarks memory-constrained configuration
func BenchmarkConnectionPoolSettings_MemoryConstrained(b *testing.B) {
	config := &RetryConfig{
		MaxIdleConns:        20,
		MaxIdleConnsPerHost: 2,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
		ExpectContinueTimeout: 500 * time.Millisecond,
	}

	benchmarkWithConfig(b, config)
}

// BenchmarkConnectionPoolSettings_Default benchmarks default configuration
func BenchmarkConnectionPoolSettings_Default(b *testing.B) {
	config := DefaultRetryConfig()
	benchmarkWithConfig(b, config)
}

// benchmarkWithConfig is a helper function for benchmarking with specific configuration
func benchmarkWithConfig(b *testing.B, config *RetryConfig) {
	// Create test server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	client := NewRetryClient(config)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, err := http.NewRequest("GET", server.URL, nil)
			if err != nil {
				b.Fatalf("Failed to create request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				b.Fatalf("Request failed: %v", err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkNewClientCreation benchmarks the overhead of creating new clients
func BenchmarkNewClientCreation(b *testing.B) {
	config := DefaultRetryConfig()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = NewRetryClient(config)
	}
}

// BenchmarkNewClientCreationWithCustomTransport benchmarks client creation with custom transport
func BenchmarkNewClientCreationWithCustomTransport(b *testing.B) {
	customTransport := &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
	}

	config := &RetryConfig{
		Transport: customTransport,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = NewRetryClient(config)
	}
}

// BenchmarkGetConnectionStats benchmarks the overhead of retrieving connection statistics
func BenchmarkGetConnectionStats(b *testing.B) {
	client := NewRetryClient(nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = client.GetConnectionStats()
	}
}

// BenchmarkCloseIdleConnections benchmarks the overhead of closing idle connections
func BenchmarkCloseIdleConnections(b *testing.B) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	client := NewRetryClient(nil)

	// Make a few requests to establish connections
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", server.URL, nil)
		resp, err := client.Do(req)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
		resp.Body.Close()
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		client.CloseIdleConnections()
	}
}

// BenchmarkTransportCreation benchmarks the overhead of creating optimized transport
func BenchmarkTransportCreation(b *testing.B) {
	config := DefaultRetryConfig()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = createOptimizedTransport(config)
	}
}
