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

package http_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	httpfilter "github.com/vedadiyan/kontrol/internal/filters/http"
	"github.com/vedadiyan/kontrol/internal/pipeline"
)

// ============================================================================
// Test Mocks and Helpers
// ============================================================================

type mockFilter struct {
	pipeline.AbstractFilter
	doFunc func(context.Context, pipeline.ResponseNode, *http.Request) error
}

func (m *mockFilter) Do(ctx context.Context, rn pipeline.ResponseNode, req *http.Request) error {
	if m.doFunc != nil {
		return m.doFunc(ctx, rn, req)
	}
	return nil
}

type mockResponseNode struct {
	current  *http.Response
	previous pipeline.ResponseNode
	next     pipeline.ResponseNode
}

func (m *mockResponseNode) Prev() pipeline.ResponseNode { return m.previous }
func (m *mockResponseNode) Next() pipeline.ResponseNode { return m.next }
func (m *mockResponseNode) Current() *http.Response     { return m.current }
func (m *mockResponseNode) Set(rs *http.Response) pipeline.ResponseNode {
	return &mockResponseNode{current: rs, previous: m}
}

func createTestServer(statusCode int, responseBody string, delay time.Duration) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if delay > 0 {
			time.Sleep(delay)
		}
		w.WriteHeader(statusCode)
		w.Write([]byte(responseBody))
	}))
}

// ============================================================================
// Public API Tests
// ============================================================================

func TestNew(t *testing.T) {
	t.Run("creates valid filter", func(t *testing.T) {
		targetURL, _ := url.Parse("https://api.example.com")
		previousFilter := &mockFilter{}

		filter := httpfilter.New("api-filter", targetURL, previousFilter)

		if filter == nil {
			t.Error("Expected non-nil filter")
		}

		// Verify it implements the Filter interface
		var _ pipeline.Filter = filter
	})

	t.Run("accepts configuration options", func(t *testing.T) {
		targetURL, _ := url.Parse("https://api.example.com")

		backgroundOpt := func(opts *pipeline.FilterOptions) {
			opts.Background = true
			opts.Timeout = 30 * time.Second
		}

		filter := httpfilter.New("api-filter", targetURL, nil, backgroundOpt)

		if filter == nil {
			t.Error("Expected non-nil filter with options")
		}
	})
}

func TestHTTPFilter_SuccessfulRequests(t *testing.T) {
	t.Run("processes 200 OK response", func(t *testing.T) {
		server := createTestServer(200, `{"status": "success"}`, 0)
		defer server.Close()

		targetURL, _ := url.Parse(server.URL)

		nextCalled := false
		mockNext := &mockFilter{
			doFunc: func(ctx context.Context, rn pipeline.ResponseNode, req *http.Request) error {
				nextCalled = true
				if rn.Current() == nil {
					t.Error("Expected response to be set")
				}
				if rn.Current().StatusCode != 200 {
					t.Errorf("Expected status 200, got %d", rn.Current().StatusCode)
				}
				return nil
			},
		}

		filter := httpfilter.New("test-filter", targetURL, nil)
		filter.OnNext(mockNext)

		ctx := context.Background()
		req := httptest.NewRequest("GET", "/test", nil)
		responseNode := &mockResponseNode{}

		err := filter.Do(ctx, responseNode, req)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !nextCalled {
			t.Error("Expected next filter to be called")
		}
	})

	t.Run("handles different 2xx status codes", func(t *testing.T) {
		statusCodes := []int{200, 201, 202, 204, 299}

		for _, statusCode := range statusCodes {
			t.Run(fmt.Sprintf("status_%d", statusCode), func(t *testing.T) {
				server := createTestServer(statusCode, "OK", 0)
				defer server.Close()

				targetURL, _ := url.Parse(server.URL)
				filter := httpfilter.New("test-filter", targetURL, nil)
				filter.OnNext(&mockFilter{})

				ctx := context.Background()
				req := httptest.NewRequest("GET", "/", nil)
				responseNode := &mockResponseNode{}

				err := filter.Do(ctx, responseNode, req)
				if err != nil {
					t.Errorf("Status %d should succeed, got error: %v", statusCode, err)
				}
			})
		}
	})
}

func TestHTTPFilter_ErrorHandling(t *testing.T) {
	t.Run("handles 4xx client errors", func(t *testing.T) {
		server := createTestServer(404, "Not Found", 0)
		defer server.Close()

		targetURL, _ := url.Parse(server.URL)

		failCalled := false
		mockFail := &mockFilter{
			doFunc: func(context.Context, pipeline.ResponseNode, *http.Request) error {
				failCalled = true
				return errors.New("failure handled")
			},
		}

		filter := httpfilter.New("test-filter", targetURL, nil)
		filter.OnFail(mockFail)

		ctx := context.Background()
		req := httptest.NewRequest("GET", "/notfound", nil)
		responseNode := &mockResponseNode{}

		err := filter.Do(ctx, responseNode, req)

		if err == nil {
			t.Error("Expected error for 404 response")
		}
		if !failCalled {
			t.Error("Expected failure filter to be called")
		}
	})

	t.Run("handles 5xx server errors", func(t *testing.T) {
		server := createTestServer(500, "Internal Server Error", 0)
		defer server.Close()

		targetURL, _ := url.Parse(server.URL)

		filter := httpfilter.New("test-filter", targetURL, nil)
		filter.OnFail(&mockFilter{})

		ctx := context.Background()
		req := httptest.NewRequest("GET", "/error", nil)
		responseNode := &mockResponseNode{}

		err := filter.Do(ctx, responseNode, req)

		if err == nil {
			t.Error("Expected error for 500 response")
		}
	})

	t.Run("handles network errors", func(t *testing.T) {
		// Use invalid URL to trigger network error
		targetURL, _ := url.Parse("http://invalid-host-that-does-not-exist.local")

		filter := httpfilter.New("test-filter", targetURL, nil)

		ctx := context.Background()
		req := httptest.NewRequest("GET", "/", nil)
		responseNode := &mockResponseNode{}

		err := filter.Do(ctx, responseNode, req)

		if err == nil {
			t.Error("Expected network error")
		}
	})
}

func TestHTTPFilter_HTTPMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			server := createTestServer(200, "OK", 0)
			defer server.Close()

			targetURL, _ := url.Parse(server.URL)
			filter := httpfilter.New("test-filter", targetURL, nil)
			filter.OnNext(&mockFilter{})

			ctx := context.Background()
			req := httptest.NewRequest(method, "/", nil)
			responseNode := &mockResponseNode{}

			err := filter.Do(ctx, responseNode, req)
			if err != nil {
				t.Errorf("Method %s failed: %v", method, err)
			}
		})
	}
}

func TestHTTPFilter_RequestHeaders(t *testing.T) {
	t.Run("preserves request headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer token123" {
				t.Error("Expected Authorization header to be preserved")
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Error("Expected Content-Type header to be preserved")
			}
			w.WriteHeader(200)
		}))
		defer server.Close()

		targetURL, _ := url.Parse(server.URL)
		filter := httpfilter.New("test-filter", targetURL, nil)
		filter.OnNext(&mockFilter{})

		ctx := context.Background()
		req := httptest.NewRequest("POST", "/api/data", nil)
		req.Header.Set("Authorization", "Bearer token123")
		req.Header.Set("Content-Type", "application/json")
		responseNode := &mockResponseNode{}

		err := filter.Do(ctx, responseNode, req)
		if err != nil {
			t.Errorf("Request with headers failed: %v", err)
		}
	})
}

func TestHTTPFilter_BackgroundExecution(t *testing.T) {
	t.Run("background execution doesn't block", func(t *testing.T) {
		server := createTestServer(200, "OK", 100*time.Millisecond)
		defer server.Close()

		targetURL, _ := url.Parse(server.URL)

		backgroundOpt := func(opts *pipeline.FilterOptions) { opts.Background = true }
		filter := httpfilter.New("bg-filter", targetURL, nil, backgroundOpt)
		filter.OnNext(&mockFilter{})

		start := time.Now()
		ctx := context.Background()
		req := httptest.NewRequest("GET", "/", nil)
		responseNode := &mockResponseNode{}

		err := filter.Do(ctx, responseNode, req)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Background execution failed: %v", err)
		}
		if duration > 50*time.Millisecond {
			t.Error("Background execution should not block")
		}
	})

	t.Run("foreground execution blocks until complete", func(t *testing.T) {
		server := createTestServer(200, "OK", 100*time.Millisecond)
		defer server.Close()

		targetURL, _ := url.Parse(server.URL)
		filter := httpfilter.New("fg-filter", targetURL, nil)
		filter.OnNext(&mockFilter{})

		start := time.Now()
		ctx := context.Background()
		req := httptest.NewRequest("GET", "/", nil)
		responseNode := &mockResponseNode{}

		err := filter.Do(ctx, responseNode, req)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Foreground execution failed: %v", err)
		}
		if duration < 90*time.Millisecond {
			t.Error("Foreground execution should block until complete")
		}
	})
}

func TestHTTPFilter_ChainIntegration(t *testing.T) {
	t.Run("integrates with filter chain", func(t *testing.T) {
		server := createTestServer(200, "API Response", 0)
		defer server.Close()

		targetURL, _ := url.Parse(server.URL)

		previous := &mockFilter{
			doFunc: func(context.Context, pipeline.ResponseNode, *http.Request) error {
				return nil
			},
		}

		nextCalled := false
		next := &mockFilter{
			doFunc: func(ctx context.Context, rn pipeline.ResponseNode, req *http.Request) error {
				nextCalled = true
				if rn.Current() == nil {
					t.Error("Expected HTTP response to be set")
				}
				return nil
			},
		}

		httpFilter := httpfilter.New("http-filter", targetURL, previous)
		httpFilter.OnNext(next)

		ctx := context.Background()
		req := httptest.NewRequest("GET", "/api", nil)
		responseNode := &mockResponseNode{}

		err := httpFilter.Do(ctx, responseNode, req)

		if err != nil {
			t.Errorf("Chain execution failed: %v", err)
		}
		if !nextCalled {
			t.Error("Expected next filter to be called")
		}
	})
}
