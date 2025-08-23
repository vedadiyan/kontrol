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
	"net/url"

	"github.com/vedadiyan/kontrol/internal/pipeline"
)

// ============================================================================
// HTTP Filter Implementation
// ============================================================================

// HTTPFilter is a pipeline filter that performs HTTP requests to a specified URL.
// It supports both synchronous and asynchronous execution based on configuration.
type HTTPFilter struct {
	pipeline.AbstractFilter
	targetURL *url.URL // The target URL for HTTP requests
}

// ============================================================================
// Constructor
// ============================================================================

// New creates a new HTTP filter with the specified target URL and
// previous filter in the chain.
func New(id string, targetURL *url.URL, previousFilter pipeline.Filter, opts ...pipeline.FilterOption) *HTTPFilter {
	filter := new(HTTPFilter)
	filter.FilterId = id
	filter.targetURL = targetURL
	filter.Previous = previousFilter

	// Apply configuration options
	for _, opt := range opts {
		opt(&filter.FilterOptions)
	}

	return filter
}

// ============================================================================
// Filter Interface Implementation
// ============================================================================

// Do executes the HTTP filter, choosing between synchronous or asynchronous
// execution based on the filter configuration.
func (f *HTTPFilter) Do(ctx context.Context, responseNode pipeline.ResponseNode, request *http.Request) error {
	if f.FilterOptions.Async {
		return f.executeAsync(ctx, responseNode, request)
	}
	return f.executeSync(ctx, responseNode, request)
}

// ============================================================================
// Synchronous Execution
// ============================================================================

// executeSync performs a synchronous HTTP request and processes the response.
func (f *HTTPFilter) executeSync(ctx context.Context, responseNode pipeline.ResponseNode, request *http.Request) error {
	response, err := performHTTPRequest(ctx, f.targetURL, request)
	if err != nil {
		return f.handleError(ctx, responseNode, request, err)
	}

	if !isSuccessStatusCode(response.StatusCode) {
		httpErr := newHTTPError(f.targetURL.String(), response.StatusCode)
		return f.handleError(ctx, responseNode, request, httpErr)
	}

	// Continue to next filter with the new response
	nextNode := responseNode.Set(response)
	return f.Next().Do(ctx, nextNode, request)
}

// ============================================================================
// Asynchronous Execution
// ============================================================================

// executeAsync performs an asynchronous HTTP request using goroutines.
func (f *HTTPFilter) executeAsync(ctx context.Context, responseNode pipeline.ResponseNode, request *http.Request) error {
	resultChannel := make(chan any)
	ctx = context.WithValue(ctx, pipeline.AsyncFilteId(f.Id()), resultChannel)

	// Execute HTTP request in background
	go func() {
		response, err := performHTTPRequest(ctx, f.targetURL, request)
		if err != nil {
			resultChannel <- f.handleError(ctx, responseNode, request, err)
			return
		}

		if !isSuccessStatusCode(response.StatusCode) {
			httpErr := newHTTPError(f.targetURL.String(), response.StatusCode)
			resultChannel <- f.handleError(ctx, responseNode, request, httpErr)
			return
		}

		// Send the new response node to the channel
		resultChannel <- responseNode.Set(response)
	}()

	// Continue immediately to next filter without waiting
	return f.Next().Do(ctx, responseNode, request)
}

// ============================================================================
// Helper Methods
// ============================================================================

// handleError processes errors by joining them with the failure filter's result.
func (f *HTTPFilter) handleError(ctx context.Context, responseNode pipeline.ResponseNode, request *http.Request, err error) error {
	if f.Fail() == nil {
		return err
	}
	failureErr := f.Fail().Do(ctx, responseNode, request)
	return errors.Join(err, failureErr)
}

// ============================================================================
// HTTP Utilities
// ============================================================================

// newHTTPError creates a formatted error for HTTP request failures.
func newHTTPError(url string, statusCode int) error {
	return fmt.Errorf("HTTP request to %s failed with status %d", url, statusCode)
}

// performHTTPRequest executes an HTTP request to the specified URL.
func performHTTPRequest(ctx context.Context, targetURL *url.URL, originalRequest *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	clonedRequest := originalRequest.Clone(ctx)
	clonedRequest.URL = targetURL
	clonedRequest.RequestURI = "" // Clear RequestURI for client requests

	return http.DefaultClient.Do(clonedRequest)
}

// isSuccessStatusCode checks if the HTTP status code indicates success (2xx range).
func isSuccessStatusCode(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}
