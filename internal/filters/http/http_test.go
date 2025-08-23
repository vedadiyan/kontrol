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
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

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
// Constructor Tests
// ============================================================================

func TestNew(t *testing.T) {
	t.Run("sets internal fields correctly", func(t *testing.T) {
		targetURL, _ := url.Parse("https://example.com")
		previousFilter := &mockFilter{}

		filter := New("test-id", targetURL, previousFilter)

		if filter.FilterId != "test-id" {
			t.Errorf("Expected FilterId 'test-id', got '%s'", filter.FilterId)
		}
		if filter.targetURL != targetURL {
			t.Error("Expected targetURL to be set correctly")
		}
		if filter.Previous != previousFilter {
			t.Error("Expected Previous filter to be set correctly")
		}

		// Verify it implements the Filter interface
		var _ pipeline.Filter = filter
	})

	t.Run("applies options to embedded struct", func(t *testing.T) {
		backgroundOpt := func(opts *pipeline.FilterOptions) {
			opts.Background = true
			opts.Timeout = 5 * time.Second
		}

		filter := New("test-id", nil, nil, backgroundOpt)

		if !filter.FilterOptions.Background {
			t.Error("Expected Background option to be true")
		}
		if filter.FilterOptions.Timeout != 5*time.Second {
			t.Errorf("Expected Timeout 5s, got %v", filter.FilterOptions.Timeout)
		}
	})
}

// ============================================================================
// Do Function Tests
// ============================================================================

func TestDo(t *testing.T) {
	t.Run("chooses foreground execution by default", func(t *testing.T) {
		server := createTestServer(200, "success", 100*time.Millisecond)
		defer server.Close()

		targetURL, _ := url.Parse(server.URL)
		filter := New("fg-filter", targetURL, nil)

		nextCalled := false
		mockNext := &mockFilter{
			doFunc: func(ctx context.Context, rn pipeline.ResponseNode, req *http.Request) error {
				nextCalled = true
				if rn.Current().StatusCode != 200 {
					t.Errorf("Expected status 200, got %d", rn.Current().StatusCode)
				}
				return nil
			},
		}
		filter.OnNext(mockNext)

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
		if !nextCalled {
			t.Error("Expected next filter to be called")
		}
	})

	t.Run("chooses background execution when configured", func(t *testing.T) {
		server := createTestServer(200, "OK", 100*time.Millisecond)
		defer server.Close()

		targetURL, _ := url.Parse(server.URL)

		backgroundOpt := func(opts *pipeline.FilterOptions) {
			opts.Background = true
		}
		filter := New("bg-filter", targetURL, nil, backgroundOpt)

		nextCalled := false
		mockNext := &mockFilter{
			doFunc: func(ctx context.Context, rn pipeline.ResponseNode, req *http.Request) error {
				nextCalled = true
				return nil
			},
		}
		filter.OnNext(mockNext)

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
		if !nextCalled {
			t.Error("Expected next filter to be called immediately")
		}
	})

	t.Run("handles different HTTP methods", func(t *testing.T) {
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

		for _, method := range methods {
			t.Run(method, func(t *testing.T) {
				server := createTestServer(200, "OK", 0)
				defer server.Close()

				targetURL, _ := url.Parse(server.URL)
				filter := New("test-filter", targetURL, nil)
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
	})

	t.Run("handles different success status codes", func(t *testing.T) {
		successCodes := []int{200, 201, 202, 204, 206, 299}

		for _, statusCode := range successCodes {
			t.Run(fmt.Sprintf("status_%d", statusCode), func(t *testing.T) {
				server := createTestServer(statusCode, "OK", 0)
				defer server.Close()

				targetURL, _ := url.Parse(server.URL)
				filter := New("test-filter", targetURL, nil)

				nextCalled := false
				mockNext := &mockFilter{
					doFunc: func(ctx context.Context, rn pipeline.ResponseNode, req *http.Request) error {
						nextCalled = true
						if rn.Current().StatusCode != statusCode {
							t.Errorf("Expected status %d, got %d", statusCode, rn.Current().StatusCode)
						}
						return nil
					},
				}
				filter.OnNext(mockNext)

				ctx := context.Background()
				req := httptest.NewRequest("GET", "/", nil)
				responseNode := &mockResponseNode{}

				err := filter.Do(ctx, responseNode, req)
				if err != nil {
					t.Errorf("Status %d should succeed, got error: %v", statusCode, err)
				}
				if !nextCalled {
					t.Errorf("Expected next filter to be called for status %d", statusCode)
				}
			})
		}
	})

	t.Run("handles client and server errors", func(t *testing.T) {
		errorCodes := []int{400, 401, 403, 404, 500, 501, 502, 503}

		for _, statusCode := range errorCodes {
			t.Run(fmt.Sprintf("status_%d", statusCode), func(t *testing.T) {
				server := createTestServer(statusCode, "Error", 0)
				defer server.Close()

				targetURL, _ := url.Parse(server.URL)

				failCalled := false
				mockFail := &mockFilter{
					doFunc: func(context.Context, pipeline.ResponseNode, *http.Request) error {
						failCalled = true
						return errors.New("failure handled")
					},
				}

				filter := New("test-filter", targetURL, nil)
				filter.OnFail(mockFail)

				ctx := context.Background()
				req := httptest.NewRequest("GET", "/", nil)
				responseNode := &mockResponseNode{}

				err := filter.Do(ctx, responseNode, req)

				if err == nil {
					t.Errorf("Expected error for %d response", statusCode)
				}
				if !failCalled {
					t.Errorf("Expected failure filter to be called for %d", statusCode)
				}
			})
		}
	})

	t.Run("handles network errors", func(t *testing.T) {
		targetURL, _ := url.Parse("http://127.0.0.1:1")
		filter := New("test-filter", targetURL, nil)

		ctx := context.Background()
		req := httptest.NewRequest("GET", "/", nil)
		responseNode := &mockResponseNode{}

		err := filter.Do(ctx, responseNode, req)
		if err == nil {
			t.Error("Expected network error")
		}
	})

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
		filter := New("test-filter", targetURL, nil)
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

// ============================================================================
// Private Method Tests
// ============================================================================

func TestExecuteForeground(t *testing.T) {
	t.Run("calls next filter on success", func(t *testing.T) {
		server := createTestServer(200, "success", 0)
		defer server.Close()

		targetURL, _ := url.Parse(server.URL)
		filter := New("test-id", targetURL, nil)

		nextCalled := false
		mockNext := &mockFilter{
			doFunc: func(ctx context.Context, rn pipeline.ResponseNode, req *http.Request) error {
				nextCalled = true
				return nil
			},
		}
		filter.OnNext(mockNext)

		ctx := context.Background()
		req := httptest.NewRequest("GET", "/", nil)
		responseNode := &mockResponseNode{}

		err := filter.executeForeground(ctx, responseNode, req)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !nextCalled {
			t.Error("Expected next filter to be called")
		}
	})

	t.Run("calls fail filter on HTTP error", func(t *testing.T) {
		server := createTestServer(500, "error", 0)
		defer server.Close()

		targetURL, _ := url.Parse(server.URL)
		filter := New("test-id", targetURL, nil)

		failCalled := false
		mockFail := &mockFilter{
			doFunc: func(context.Context, pipeline.ResponseNode, *http.Request) error {
				failCalled = true
				return errors.New("failure handled")
			},
		}
		filter.OnFail(mockFail)

		ctx := context.Background()
		req := httptest.NewRequest("GET", "/", nil)
		responseNode := &mockResponseNode{}

		err := filter.executeForeground(ctx, responseNode, req)

		if err == nil {
			t.Error("Expected error for 500 response")
		}
		if !failCalled {
			t.Error("Expected failure filter to be called")
		}
	})

	t.Run("handles network errors", func(t *testing.T) {
		targetURL, _ := url.Parse("http://127.0.0.1:1")
		filter := New("test-id", targetURL, nil)

		failCalled := false
		mockFail := &mockFilter{
			doFunc: func(context.Context, pipeline.ResponseNode, *http.Request) error {
				failCalled = true
				return errors.New("network failure handled")
			},
		}
		filter.OnFail(mockFail)

		ctx := context.Background()
		req := httptest.NewRequest("GET", "/", nil)
		responseNode := &mockResponseNode{}

		err := filter.executeForeground(ctx, responseNode, req)

		if err == nil {
			t.Error("Expected network error")
		}
		if !failCalled {
			t.Error("Expected failure filter to be called for network error")
		}
	})
}

func TestExecuteBackground(t *testing.T) {
	t.Run("returns immediately on success", func(t *testing.T) {
		server := createTestServer(200, "success", 100*time.Millisecond)
		defer server.Close()

		targetURL, _ := url.Parse(server.URL)
		filter := New("test-id", targetURL, nil)

		mockNext := &mockFilter{}
		filter.OnNext(mockNext)

		ctx := context.Background()
		req := httptest.NewRequest("GET", "/", nil)
		responseNode := &mockResponseNode{}

		start := time.Now()
		err := filter.executeBackground(ctx, responseNode, req)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if duration > 10*time.Millisecond {
			t.Error("Should return immediately")
		}

		// Wait for goroutine to complete and verify channel gets response
		channelValue := ctx.Value(pipeline.BackgroundFilterId("test-id"))
		if channel, ok := channelValue.(chan any); ok {
			select {
			case result := <-channel:
				if responseNode, ok := result.(pipeline.ResponseNode); ok {
					if responseNode.Current() == nil {
						t.Error("Expected response to be set in channel")
					}
				} else {
					t.Error("Expected ResponseNode in channel")
				}
			case <-time.After(200 * time.Millisecond):
				t.Error("Expected result in channel within timeout")
			}
		}
	})

	t.Run("handles HTTP errors in goroutine", func(t *testing.T) {
		server := createTestServer(404, "Not Found", 0)
		defer server.Close()

		targetURL, _ := url.Parse(server.URL)
		filter := New("test-id", targetURL, nil)

		failCalled := false
		mockFail := &mockFilter{
			doFunc: func(context.Context, pipeline.ResponseNode, *http.Request) error {
				failCalled = true
				return errors.New("failure handled")
			},
		}
		filter.OnFail(mockFail)

		mockNext := &mockFilter{}
		filter.OnNext(mockNext)

		ctx := context.Background()
		req := httptest.NewRequest("GET", "/", nil)
		responseNode := &mockResponseNode{}

		err := filter.executeBackground(ctx, responseNode, req)
		if err != nil {
			t.Errorf("Background execution should not return error immediately, got %v", err)
		}

		// Wait for the goroutine to complete and check the channel
		channelValue := ctx.Value(pipeline.BackgroundFilterId("test-id"))
		if channel, ok := channelValue.(chan any); ok {
			select {
			case result := <-channel:
				if resultErr, isError := result.(error); isError {
					if resultErr == nil {
						t.Error("Expected error result from channel")
					}
				} else {
					t.Error("Expected error result from channel, got something else")
				}
			case <-time.After(100 * time.Millisecond):
				t.Error("Expected result in channel within timeout")
			}
		}

		// Give the goroutine time to call handleError
		time.Sleep(50 * time.Millisecond)

		if !failCalled {
			t.Error("Expected failure filter to be called in background goroutine")
		}
	})

	t.Run("handles network errors in goroutine", func(t *testing.T) {
		targetURL, _ := url.Parse("http://127.0.0.1:1")
		filter := New("test-id", targetURL, nil)

		failCalled := false
		mockFail := &mockFilter{
			doFunc: func(context.Context, pipeline.ResponseNode, *http.Request) error {
				failCalled = true
				return errors.New("network failure handled")
			},
		}
		filter.OnFail(mockFail)

		mockNext := &mockFilter{}
		filter.OnNext(mockNext)

		ctx := context.Background()
		req := httptest.NewRequest("GET", "/", nil)
		responseNode := &mockResponseNode{}

		err := filter.executeBackground(ctx, responseNode, req)
		if err != nil {
			t.Errorf("Background execution should not return error immediately, got %v", err)
		}

		// Wait for the goroutine to complete and check the channel
		channelValue := ctx.Value(pipeline.BackgroundFilterId("test-id"))
		if channel, ok := channelValue.(chan any); ok {
			select {
			case result := <-channel:
				if _, isError := result.(error); !isError {
					t.Error("Expected error result from channel")
				}
			case <-time.After(1 * time.Second):
				t.Error("Expected result in channel within timeout")
			}
		}

		// Give the goroutine time to call handleError
		time.Sleep(200 * time.Millisecond)

		if !failCalled {
			t.Error("Expected failure filter to be called for network error in background")
		}
	})

	t.Run("sets context value correctly", func(t *testing.T) {
		server := createTestServer(200, "success", 0)
		defer server.Close()

		targetURL, _ := url.Parse(server.URL)
		filter := New("bg-test", targetURL, nil)

		contextChecked := false
		mockNext := &mockFilter{
			doFunc: func(ctx context.Context, rn pipeline.ResponseNode, req *http.Request) error {
				if ctx.Value(pipeline.BackgroundFilterId("bg-test")) == nil {
					t.Error("Expected context to contain background filter channel")
				}
				contextChecked = true
				return nil
			},
		}
		filter.OnNext(mockNext)

		ctx := context.Background()
		req := httptest.NewRequest("GET", "/", nil)
		responseNode := &mockResponseNode{}

		err := filter.executeBackground(ctx, responseNode, req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !contextChecked {
			t.Error("Expected next filter to be called to verify context")
		}
	})
}

func TestHandleError(t *testing.T) {
	t.Run("returns original error when no fail filter", func(t *testing.T) {
		filter := New("test-id", nil, nil)

		ctx := context.Background()
		responseNode := &mockResponseNode{}
		req := httptest.NewRequest("GET", "/", nil)
		originalErr := errors.New("test error")

		err := filter.handleError(ctx, responseNode, req, originalErr)

		if err != originalErr {
			t.Error("Expected original error when no failure filter is set")
		}
	})

	t.Run("joins errors correctly", func(t *testing.T) {
		filter := New("test-id", nil, nil)

		mockFail := &mockFilter{
			doFunc: func(context.Context, pipeline.ResponseNode, *http.Request) error {
				return errors.New("failure error")
			},
		}
		filter.OnFail(mockFail)

		ctx := context.Background()
		responseNode := &mockResponseNode{}
		req := httptest.NewRequest("GET", "/", nil)
		originalErr := errors.New("original error")

		err := filter.handleError(ctx, responseNode, req, originalErr)

		if err == nil {
			t.Error("Expected joined error")
		}

		errStr := err.Error()
		if !strings.Contains(errStr, "original error") || !strings.Contains(errStr, "failure error") {
			t.Errorf("Expected joined error message, got %s", errStr)
		}
	})
}

// ============================================================================
// Private Function Tests
// ============================================================================

func TestIsSuccessStatusCode(t *testing.T) {
	tests := []struct {
		code     int
		expected bool
	}{
		{100, false},
		{199, false},
		{200, true},
		{201, true},
		{299, true},
		{300, false},
		{404, false},
		{500, false},
	}

	for _, test := range tests {
		result := isSuccessStatusCode(test.code)
		if result != test.expected {
			t.Errorf("isSuccessStatusCode(%d): expected %v, got %v",
				test.code, test.expected, result)
		}
	}
}

func TestPerformHTTPRequest(t *testing.T) {
	t.Run("clones request without modifying original", func(t *testing.T) {
		server := createTestServer(200, "test", 0)
		defer server.Close()

		targetURL, _ := url.Parse(server.URL)
		originalReq := httptest.NewRequest("GET", "http://original.com/path", nil)
		originalReq.Header.Set("Test-Header", "test-value")
		originalURL := originalReq.URL.String()
		originalURI := originalReq.RequestURI

		ctx := context.Background()
		resp, err := performHTTPRequest(ctx, targetURL, originalReq)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if resp == nil {
			t.Error("Expected response, got nil")
		}

		if originalReq.URL.String() != originalURL {
			t.Error("Original request URL was modified")
		}
		if originalReq.RequestURI != originalURI {
			t.Error("Original request RequestURI was modified")
		}
	})

	t.Run("preserves headers in cloned request", func(t *testing.T) {
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
		originalReq := httptest.NewRequest("POST", "http://original.com", nil)
		originalReq.Header.Set("Authorization", "Bearer token123")
		originalReq.Header.Set("Content-Type", "application/json")

		ctx := context.Background()
		resp, err := performHTTPRequest(ctx, targetURL, originalReq)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("sets target URL correctly", func(t *testing.T) {
		receivedURL := ""
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedURL = r.URL.String()
			w.WriteHeader(200)
		}))
		defer server.Close()

		targetURL, _ := url.Parse(server.URL + "/api/endpoint")
		originalReq := httptest.NewRequest("GET", "http://original.com/different/path", nil)

		ctx := context.Background()
		_, err := performHTTPRequest(ctx, targetURL, originalReq)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !strings.Contains(receivedURL, "/api/endpoint") {
			t.Errorf("Expected target URL to be used, got %s", receivedURL)
		}
	})
}

func TestNewHTTPError(t *testing.T) {
	t.Run("formats error message correctly", func(t *testing.T) {
		err := newHTTPError("https://example.com/api", 404)

		expected := "HTTP request to https://example.com/api failed with status 404"
		if err.Error() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("handles different status codes", func(t *testing.T) {
		testCases := []struct {
			url        string
			statusCode int
			statusStr  string
		}{
			{"https://api.example.com", 400, "400"},
			{"http://localhost:8080/endpoint", 500, "500"},
			{"https://service.com/api/v1", 503, "503"},
		}

		for _, tc := range testCases {
			err := newHTTPError(tc.url, tc.statusCode)
			errMsg := err.Error()

			if !strings.Contains(errMsg, tc.url) {
				t.Errorf("Expected error to contain URL %s", tc.url)
			}
			if !strings.Contains(errMsg, tc.statusStr) {
				t.Errorf("Expected error to contain status code %s, got: %s", tc.statusStr, errMsg)
			}
		}
	})
}
