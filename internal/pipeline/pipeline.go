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

// Package pipeline provides a flexible HTTP request/response processing framework
// that allows chaining of filters with support for background execution, failure
// handling, and response navigation through a doubly-linked response chain.
package pipeline

import (
	"context"
	"io"
	"net/http"
	"time"
)

// BackgroundFilterId represents a unique identifier for background filters,
// allowing tracking and management of filters executing asynchronously.
type BackgroundFilterId string

// Response wraps an HTTP response with its body, headers, and trailers
// in a format suitable for pipeline processing and navigation.
type Response struct {
	Body       []byte      // The response body content
	StatusCode int         // HTTP status code (e.g., 200, 404, 500)
	Header     http.Header // HTTP response headers
	Trailer    http.Header // HTTP response trailers
}

// ============================================================================
// Core Interfaces
// ============================================================================

// ResponseNode represents a node in the response chain, allowing navigation
// through previous and next responses while maintaining the current HTTP response.
type ResponseNode interface {
	// Prev returns the previous ResponseNode in the chain, or nil if this is the first node.
	Prev() ResponseNode
	// Next returns the next ResponseNode in the chain, or nil if this is the last node.
	Next() ResponseNode
	// Current returns the HTTP response stored in this node.
	Current() *Response
	// Set updates the current HTTP response and creates a new next node,
	// returning the newly created node for continued chaining.
	Set(rs *Response) ResponseNode
}

// Filter defines a processing unit in the pipeline that can be chained
// with other filters and process HTTP requests/responses.
type Filter interface {
	// Prev returns the previous filter in the chain, enabling backward navigation.
	Prev() Filter
	// Do processes the HTTP request and response, potentially modifying the ResponseNode.
	// It receives a context for cancellation/timeout, the current response node,
	// and the original HTTP request.
	Do(context.Context, ResponseNode, *http.Request) error
	// Id returns a unique identifier for this filter instance.
	Id() string
}

// ============================================================================
// Filter Configuration
// ============================================================================

// FilterOptions holds configuration options for filter creation and setup.
type FilterOptions struct {
	Background bool          // Whether the filter should execute in the background
	Timeout    time.Duration // Maximum execution time for the filter
}

// FilterOption is a function type for configuring FilterOptions using the
// functional options pattern.
type FilterOption func(*FilterOptions)

// ============================================================================
// Filter Implementation
// ============================================================================

// AbstractFilter provides base functionality for filters including chaining,
// failure handling, and navigation. It embeds the Filter interface but leaves
// the Do method to be implemented by concrete filter types.
type AbstractFilter struct {
	Filter               // Embedded Filter interface to be implemented by concrete types
	FilterOptions        // Embedded configuration options
	FilterId      string // Unique identifier for this filter instance
	Previous      Filter // The previous filter in the chain (exported for external setup)
	onFail        Filter // Filter to execute on failure
	onNext        Filter // Filter to execute next in the chain
}

// ============================================================================
// Response Node Implementation
// ============================================================================

// responseNode is the concrete implementation of ResponseNode providing
// a doubly-linked list structure for response navigation.
type responseNode struct {
	previous ResponseNode // Reference to the previous node in the chain
	next     ResponseNode // Reference to the next node in the chain
	current  *Response    // The HTTP response stored in this node
}

// ============================================================================
// Response Node Constructor
// ============================================================================

// DefaultResponseNode creates a new ResponseNode with the given HTTP response.
// This serves as the starting point for building a response chain.
func DefaultResponseNode(r *Response) ResponseNode {
	out := new(responseNode)
	out.current = r
	return out
}

// ============================================================================
// ResponseNode Methods
// ============================================================================

// Prev returns the previous ResponseNode in the chain.
func (r *responseNode) Prev() ResponseNode {
	return r.previous
}

// Next returns the next ResponseNode in the chain.
func (r *responseNode) Next() ResponseNode {
	return r.next
}

// Current returns the HTTP response stored in this node.
func (r *responseNode) Current() *Response {
	return r.current
}

// Set updates the current HTTP response and creates a new next node,
// returning the newly created node.
func (r *responseNode) Set(rs *Response) ResponseNode {
	r.current = rs
	r.next = &responseNode{r, nil, nil}
	return r.next
}

// ============================================================================
// AbstractFilter Methods
// ============================================================================

// Id returns the filter identifier.
func (f *AbstractFilter) Id() string {
	return f.FilterId
}

// Prev returns the previous filter in the chain.
func (f *AbstractFilter) Prev() Filter {
	return f.Previous
}

// OnFail sets the filter to execute when this filter fails.
// This enables custom error handling and recovery strategies.
func (f *AbstractFilter) OnFail(l Filter) {
	f.onFail = l
}

// OnNext sets the next filter to execute in the chain.
// This method is used to build the processing pipeline.
func (f *AbstractFilter) OnNext(l Filter) {
	f.onNext = l
}

// Fail returns the filter configured to handle failures.
// Returns nil if no failure handler is configured.
func (f *AbstractFilter) Fail() Filter {
	return f.onFail
}

// Next returns the next filter in the processing chain.
// Returns nil if this is the last filter in the chain.
func (f *AbstractFilter) Next() Filter {
	return f.onNext
}

// ============================================================================
// Utility Functions
// ============================================================================

// ToResponse converts an http.Response to a pipeline Response by reading
// the body and copying headers and trailers. The original response body
// is closed after reading.
//
// Returns an error if the response body cannot be read.
func ToResponse(rs http.Response) (*Response, error) {
	body, err := io.ReadAll(rs.Body)
	if err != nil {
		return nil, err
	}
	defer rs.Body.Close()
	out := new(Response)
	out.StatusCode = rs.StatusCode
	out.Body = body
	out.Header = rs.Header.Clone()
	out.Trailer = rs.Trailer.Clone()
	return out, nil
}
