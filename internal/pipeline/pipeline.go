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

type Nexter interface {
	Next() Filter
}

type Failer interface {
	Fail() Filter
}

type ResponseNode interface {
	Prev() ResponseNode
	Next() ResponseNode
	Current() *Response
	Set(rs *Response) ResponseNode
	Clone() ResponseNode
}

type Filter interface {
	Do(context.Context, ResponseNode, *http.Request) error
	Id() string
}

type FilterOptions struct {
	Timeout time.Duration
}

type FilterOption func(*FilterOptions)

type FilterWrapper struct {
	Filter
	FilterOptions
	FilterId string
	do       func(context.Context, ResponseNode, *http.Request) error
	onFail   Filter
	onNext   Filter
}

type responseNode struct {
	previous ResponseNode
	next     ResponseNode
	current  *Response
}

const (
	CONTEXT_ERR  ContextKey = "error"
	CONTEXT_PREV ContextKey = "previous"
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

func (f *FilterWrapper) Id() string {
	return f.FilterId
}

func (f *FilterWrapper) OnFail(l Filter) Filter {
	f.onFail = l
	return f
}

func (f *FilterWrapper) OnNext(l Filter) Filter {
	f.onNext = l
	return f
}

func (f *FilterWrapper) Fail() Filter {
	return f.onFail
}

func (f *FilterWrapper) Next() Filter {
	return f.onNext
}

func (f *FilterWrapper) Do(ctx context.Context, rs ResponseNode, rq *http.Request) error {
	ctx = context.WithValue(ctx, CONTEXT_PREV, f)
	return f.do(ctx, rs, rq)
}

func (f *FilterWrapper) HandleError(ctx context.Context, responseNode ResponseNode, request *http.Request, err error) error {
	if f.Fail() == nil {
		return err
	}
	ctx = context.WithValue(ctx, CONTEXT_ERR, err)
	return errors.Join(err, f.Fail().Do(ctx, responseNode, request))
}

func (f *FilterWrapper) HandleNext(ctx context.Context, responseNode ResponseNode, request *http.Request) error {
	if f.Next() == nil {
		return nil
	}
	return f.Next().Do(ctx, responseNode, request)
}

func (f *FilterWrapper) Handler(fn func(context.Context, ResponseNode, *http.Request) error) {
	f.do = fn
}

func (f *FilterWrapper) PreviousFilter(ctx context.Context) Filter {
	if filter := ctx.Value(CONTEXT_PREV); filter != nil {
		return filter.(Filter)
	}
	return nil
}

func (f *FilterWrapper) PreviousError(ctx context.Context) error {
	if filter := ctx.Value(CONTEXT_ERR); filter != nil {
		return filter.(error)
	}
	return nil
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
