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

package pipeline

import (
	"context"
	"io"
	"net/http"
	"time"
)

type BackgroundFilterId string

type Response struct {
	Body    []byte
	Header  http.Header
	Trailer http.Header
}

// ============================================================================
// Core Interfaces
// ============================================================================

// ResponseNode represents a node in the response chain, allowing navigation
// through previous and next responses while maintaining the current HTTP response.
type ResponseNode interface {
	Prev() ResponseNode
	Next() ResponseNode
	Current() *Response
	Set(rs *Response) ResponseNode
}

// Filter defines a processing unit in the pipeline that can be chained
// with other filters and process HTTP requests/responses.
type Filter interface {
	Prev() Filter
	Do(context.Context, ResponseNode, *http.Request) error
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
	Filter
	FilterOptions
	FilterId string
	Previous Filter // The previous filter in the chain (exported for external setup)
	onFail   Filter // Filter to execute on failure
	onNext   Filter // Filter to execute next in the chain
}

// ============================================================================
// Response Node Implementation
// ============================================================================

// responseNode is the concrete implementation of ResponseNode providing
// a doubly-linked list structure for response navigation.
type responseNode struct {
	previous ResponseNode
	next     ResponseNode
	current  *Response
}

// ============================================================================
// Response Node Constructor
// ============================================================================

// DefaultResponseNode creates a new ResponseNode with the given HTTP response.
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

// Id returns the filter Id.
func (f *AbstractFilter) Id() string {
	return f.FilterId
}

// Prev returns the previous filter in the chain.
func (f *AbstractFilter) Prev() Filter {
	return f.Previous
}

// OnFail sets the filter to execute when this filter fails.
func (f *AbstractFilter) OnFail(l Filter) {
	f.onFail = l
}

// OnNext sets the next filter to execute in the chain.
func (f *AbstractFilter) OnNext(l Filter) {
	f.onNext = l
}

// Fail returns the filter configured to handle failures.
func (f *AbstractFilter) Fail() Filter {
	return f.onFail
}

// Next returns the next filter in the processing chain.
func (f *AbstractFilter) Next() Filter {
	return f.onNext
}

func ToResponse(rs http.Response) (*Response, error) {
	body, err := io.ReadAll(rs.Body)
	if err != nil {
		return nil, err
	}
	defer rs.Body.Close()
	out := new(Response)
	out.Body = body
	out.Header = rs.Header.Clone()
	out.Trailer = rs.Trailer.Clone()
	return out, nil
}
