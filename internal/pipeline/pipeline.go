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
	"errors"
	"io"
	"net/http"
	"time"
)

type ContextKey string

type BackgroundFilterId string

type Response struct {
	Body       []byte      // The response body content
	StatusCode int         // HTTP status code (e.g., 200, 404, 500)
	Header     http.Header // HTTP response headers
	Trailer    http.Header // HTTP response trailers
}

type ResponseNode interface {
	Prev() ResponseNode
	Next() ResponseNode
	Current() *Response
	Set(rs *Response) ResponseNode
	Clone() ResponseNode
}

type Filter interface {
	Prev() Filter
	Do(context.Context, ResponseNode, *http.Request) error
	Id() string
}

type FilterOptions struct {
	Timeout time.Duration
}

type FilterOption func(*FilterOptions)

type AbstractFilter struct {
	Filter
	FilterOptions
	FilterId string
	Previous Filter
	onFail   Filter
	onNext   Filter
}

type responseNode struct {
	previous ResponseNode
	next     ResponseNode
	current  *Response
}

const (
	CONTEXT_ERR ContextKey = "error"
)

func DefaultResponseNode(r *Response) ResponseNode {
	out := new(responseNode)
	out.current = r
	return out
}

func (r *responseNode) Prev() ResponseNode {
	return r.previous
}

func (r *responseNode) Next() ResponseNode {
	return r.next
}

func (r *responseNode) Current() *Response {
	return r.current
}

func (r *responseNode) Set(rs *Response) ResponseNode {
	r.current = rs
	r.next = &responseNode{r, nil, nil}
	return r.next
}

func (r *responseNode) Clone() ResponseNode {
	v := *r
	return &v
}

func (f *AbstractFilter) Id() string {
	return f.FilterId
}

func (f *AbstractFilter) Prev() Filter {
	return f.Previous
}

func (f *AbstractFilter) OnFail(l Filter) {
	f.onFail = l
}

func (f *AbstractFilter) OnNext(l Filter) {
	f.onNext = l
}

func (f *AbstractFilter) Fail() Filter {
	return f.onFail
}

func (f *AbstractFilter) Next() Filter {
	return f.onNext
}

func (f *AbstractFilter) HandleError(ctx context.Context, responseNode ResponseNode, request *http.Request, err error) error {
	if f.Fail() == nil {
		return err
	}
	ctx = context.WithValue(ctx, CONTEXT_ERR, err)
	return errors.Join(err, f.Fail().Do(ctx, responseNode, request))
}

func (f *AbstractFilter) HandleNext(ctx context.Context, responseNode ResponseNode, request *http.Request) error {
	if f.Next() == nil {
		return nil
	}
	return f.Next().Do(ctx, responseNode, request)
}

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
