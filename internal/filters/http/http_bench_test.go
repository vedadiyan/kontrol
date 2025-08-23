// Copyright (c) 2025 Pouya Vedadiyan. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/vedadiyan/kontrol/internal/pipeline"
)

// ============================================================================
// Benchmark Helpers
// ============================================================================

type benchmarkFilter struct {
	pipeline.AbstractFilter
}

func (b *benchmarkFilter) Do(ctx context.Context, rn pipeline.ResponseNode, req *http.Request) error {
	return nil
}

type benchmarkResponseNode struct {
	current *http.Response
}

func (b *benchmarkResponseNode) Prev() pipeline.ResponseNode { return nil }
func (b *benchmarkResponseNode) Next() pipeline.ResponseNode { return nil }
func (b *benchmarkResponseNode) Current() *http.Response     { return b.current }
func (b *benchmarkResponseNode) Set(rs *http.Response) pipeline.ResponseNode {
	return &benchmarkResponseNode{current: rs}
}

func createBenchmarkServer(statusCode int, responseSize int, delay time.Duration) *httptest.Server {
	response := make([]byte, responseSize)
	for i := range response {
		response[i] = 'a' // Fill with 'a' characters
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if delay > 0 {
			time.Sleep(delay)
		}
		w.WriteHeader(statusCode)
		w.Write(response)
	}))
}

// ============================================================================
// Constructor Benchmarks
// ============================================================================

func BenchmarkNew(b *testing.B) {
	targetURL, _ := url.Parse("https://api.example.com")
	previousFilter := &benchmarkFilter{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = New("benchmark-filter", targetURL, previousFilter)
	}
}

func BenchmarkNewWithOptions(b *testing.B) {
	targetURL, _ := url.Parse("https://api.example.com")
	previousFilter := &benchmarkFilter{}

	backgroundOpt := func(opts *pipeline.FilterOptions) {
		opts.Background = true
		opts.Timeout = 30 * time.Second
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = New("benchmark-filter", targetURL, previousFilter, backgroundOpt)
	}
}

// ============================================================================
// Foreground Execution Benchmarks
// ============================================================================

func BenchmarkExecuteForeground_Success(b *testing.B) {
	server := createBenchmarkServer(200, 1024, 0) // 1KB response
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	filter := New("benchmark-filter", targetURL, nil)
	filter.OnNext(&benchmarkFilter{})

	ctx := context.Background()
	req := httptest.NewRequest("GET", "/", nil)
	responseNode := &benchmarkResponseNode{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		filter.executeForeground(ctx, responseNode, req)
	}
}

func BenchmarkExecuteForeground_SmallResponse(b *testing.B) {
	server := createBenchmarkServer(200, 100, 0) // 100 bytes
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	filter := New("benchmark-filter", targetURL, nil)
	filter.OnNext(&benchmarkFilter{})

	ctx := context.Background()
	req := httptest.NewRequest("GET", "/", nil)
	responseNode := &benchmarkResponseNode{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		filter.executeForeground(ctx, responseNode, req)
	}
}

func BenchmarkExecuteForeground_LargeResponse(b *testing.B) {
	server := createBenchmarkServer(200, 10*1024, 0) // 10KB response
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	filter := New("benchmark-filter", targetURL, nil)
	filter.OnNext(&benchmarkFilter{})

	ctx := context.Background()
	req := httptest.NewRequest("GET", "/", nil)
	responseNode := &benchmarkResponseNode{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		filter.executeForeground(ctx, responseNode, req)
	}
}

func BenchmarkExecuteForeground_WithLatency(b *testing.B) {
	server := createBenchmarkServer(200, 1024, 10*time.Millisecond) // 10ms latency
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	filter := New("benchmark-filter", targetURL, nil)
	filter.OnNext(&benchmarkFilter{})

	ctx := context.Background()
	req := httptest.NewRequest("GET", "/", nil)
	responseNode := &benchmarkResponseNode{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		filter.executeForeground(ctx, responseNode, req)
	}
}

// ============================================================================
// Background Execution Benchmarks
// ============================================================================

func BenchmarkExecuteBackground_Success(b *testing.B) {
	server := createBenchmarkServer(200, 1024, 0) // 1KB response
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	filter := New("benchmark-filter", targetURL, nil)
	filter.OnNext(&benchmarkFilter{})

	ctx := context.Background()
	req := httptest.NewRequest("GET", "/", nil)
	responseNode := &benchmarkResponseNode{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		filter.executeBackground(ctx, responseNode, req)
	}
}

func BenchmarkExecuteBackground_WithLatency(b *testing.B) {
	server := createBenchmarkServer(200, 1024, 10*time.Millisecond) // 10ms latency
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	filter := New("benchmark-filter", targetURL, nil)
	filter.OnNext(&benchmarkFilter{})

	ctx := context.Background()
	req := httptest.NewRequest("GET", "/", nil)
	responseNode := &benchmarkResponseNode{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		filter.executeBackground(ctx, responseNode, req)
	}
}

// ============================================================================
// Public API Benchmarks
// ============================================================================

func BenchmarkDo_Foreground(b *testing.B) {
	server := createBenchmarkServer(200, 1024, 0)
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	filter := New("benchmark-filter", targetURL, nil)
	filter.OnNext(&benchmarkFilter{})

	ctx := context.Background()
	req := httptest.NewRequest("GET", "/", nil)
	responseNode := &benchmarkResponseNode{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		filter.Do(ctx, responseNode, req)
	}
}

func BenchmarkDo_Background(b *testing.B) {
	server := createBenchmarkServer(200, 1024, 0)
	defer server.Close()

	backgroundOpt := func(opts *pipeline.FilterOptions) {
		opts.Background = true
	}

	targetURL, _ := url.Parse(server.URL)
	filter := New("benchmark-filter", targetURL, nil, backgroundOpt)
	filter.OnNext(&benchmarkFilter{})

	ctx := context.Background()
	req := httptest.NewRequest("GET", "/", nil)
	responseNode := &benchmarkResponseNode{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		filter.Do(ctx, responseNode, req)
	}
}

// ============================================================================
// HTTP Utilities Benchmarks
// ============================================================================

func BenchmarkPerformHTTPRequest(b *testing.B) {
	server := createBenchmarkServer(200, 1024, 0)
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		performHTTPRequest(ctx, targetURL, req)
	}
}

func BenchmarkPerformHTTPRequest_WithHeaders(b *testing.B) {
	server := createBenchmarkServer(200, 1024, 0)
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("User-Agent", "HTTPFilter/1.0")
	req.Header.Set("X-Request-ID", "benchmark-request")

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		performHTTPRequest(ctx, targetURL, req)
	}
}

func BenchmarkIsSuccessStatusCode(b *testing.B) {
	statusCodes := []int{200, 201, 404, 500, 302, 204, 400, 503}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		code := statusCodes[i%len(statusCodes)]
		isSuccessStatusCode(code)
	}
}

func BenchmarkNewHTTPError(b *testing.B) {
	url := "https://api.example.com/endpoint"
	statusCode := 404

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		newHTTPError(url, statusCode)
	}
}

// ============================================================================
// Error Handling Benchmarks
// ============================================================================

func BenchmarkHandleError_NoFailFilter(b *testing.B) {
	filter := New("benchmark-filter", nil, nil)

	ctx := context.Background()
	responseNode := &benchmarkResponseNode{}
	req := httptest.NewRequest("GET", "/", nil)
	err := newHTTPError("https://example.com", 404)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		filter.handleError(ctx, responseNode, req, err)
	}
}

func BenchmarkHandleError_WithFailFilter(b *testing.B) {
	filter := New("benchmark-filter", nil, nil)
	filter.OnFail(&benchmarkFilter{})

	ctx := context.Background()
	responseNode := &benchmarkResponseNode{}
	req := httptest.NewRequest("GET", "/", nil)
	err := newHTTPError("https://example.com", 404)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		filter.handleError(ctx, responseNode, req, err)
	}
}

// ============================================================================
// Comparative Benchmarks
// ============================================================================

func BenchmarkForegroundVsBackground(b *testing.B) {
	server := createBenchmarkServer(200, 1024, 5*time.Millisecond) // 5ms latency
	defer server.Close()

	targetURL, _ := url.Parse(server.URL)

	b.Run("Foreground", func(b *testing.B) {
		filter := New("fg-benchmark", targetURL, nil)
		filter.OnNext(&benchmarkFilter{})

		ctx := context.Background()
		req := httptest.NewRequest("GET", "/", nil)
		responseNode := &benchmarkResponseNode{}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			filter.Do(ctx, responseNode, req)
		}
	})

	b.Run("Background", func(b *testing.B) {
		backgroundOpt := func(opts *pipeline.FilterOptions) {
			opts.Background = true
		}

		filter := New("bg-benchmark", targetURL, nil, backgroundOpt)
		filter.OnNext(&benchmarkFilter{})

		ctx := context.Background()
		req := httptest.NewRequest("GET", "/", nil)
		responseNode := &benchmarkResponseNode{}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			filter.Do(ctx, responseNode, req)
		}
	})
}

func BenchmarkResponseSizes(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"100B", 100},
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			server := createBenchmarkServer(200, size.size, 0)
			defer server.Close()

			targetURL, _ := url.Parse(server.URL)
			filter := New("size-benchmark", targetURL, nil)
			filter.OnNext(&benchmarkFilter{})

			ctx := context.Background()
			req := httptest.NewRequest("GET", "/", nil)
			responseNode := &benchmarkResponseNode{}

			b.ResetTimer()
			b.SetBytes(int64(size.size))

			for i := 0; i < b.N; i++ {
				filter.Do(ctx, responseNode, req)
			}
		})
	}
}

func BenchmarkLatencyImpact(b *testing.B) {
	latencies := []struct {
		name  string
		delay time.Duration
	}{
		{"0ms", 0},
		{"1ms", 1 * time.Millisecond},
		{"5ms", 5 * time.Millisecond},
		{"10ms", 10 * time.Millisecond},
		{"50ms", 50 * time.Millisecond},
	}

	for _, lat := range latencies {
		b.Run(lat.name, func(b *testing.B) {
			server := createBenchmarkServer(200, 1024, lat.delay)
			defer server.Close()

			targetURL, _ := url.Parse(server.URL)
			filter := New("latency-benchmark", targetURL, nil)
			filter.OnNext(&benchmarkFilter{})

			ctx := context.Background()
			req := httptest.NewRequest("GET", "/", nil)
			responseNode := &benchmarkResponseNode{}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				filter.Do(ctx, responseNode, req)
			}
		})
	}
}
